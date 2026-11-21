// EssieuPilot — ATS 1.62.2.2 experimental physical steering DLL.
// In-game motion and Convoy synchronization have NOT been validated.
// Loads as a standard SCS telemetry plugin; the hook is never installed
// until an explicitly armed, fresh command arrives from the user app.
// No TruckersMP support. Only one attached trailer unit is supported.
#define WIN32_LEAN_AND_MEAN
#define NOMINMAX
#include <Windows.h>
#include <scssdk_telemetry.h>
#include <amtrucks/scssdk_ats.h>
#include <MinHook.h>
#include <atomic>
#include <cmath>
#include <cstddef>
#include <cstdint>
#include <cstring>
#include <cstdio>
#include <limits>

#include "../bridge/ATS162ExactBuildGate.hpp"
#include "../bridge/EssieuPilotBridge.hpp"

namespace
{
using AdvanceFn = std::uint64_t(__fastcall*)(void*);
using ApplyWheelsFn = void(__fastcall*)(void*, float);
constexpr std::uintptr_t kSteeringAdvanceRva = 0x77B140;
constexpr std::uintptr_t kApplyWheelsRva = 0x84D940;
constexpr std::uintptr_t kVirtualTableRvas[] = {
    0x22D5E88, 0x22F9F50, 0x23325D8
};
// Two layouts remain CANDIDATES even on an exact 1.62.2.2 EXE.
// TS Extra Utilities documents slot 73 and the vehicle_shared_u
// steering/wheel_data fields 0x4E8/0x4F0; Original.12's candidate
// used slot 78 and 0x6B8/0x6C0. Never write without a validated vtable
// match AND accessible data/pointer checks.
constexpr std::size_t kSourceSlot = 73;
constexpr std::size_t kLegacySlot = 78;
constexpr std::size_t kSourceAngleOffset = 0x4E8;
constexpr std::size_t kSourceWheelPtrOffset = 0x4F0;
constexpr std::size_t kLegacyAngleOffset = 0x6B8;
constexpr std::size_t kLegacyWheelPtrOffset = 0x6C0;

std::atomic<bool> g_paused{true};
std::atomic<bool> g_possibleConvoy{false};
std::atomic<bool> g_hookEnabled{false};
std::atomic<bool> g_shuttingDown{false};
AdvanceFn g_originalAdvance = nullptr;
ApplyWheelsFn g_applyWheels = nullptr;
void* g_hookAddress = nullptr;
std::uintptr_t g_imageBase = 0;
scs_log_t g_log = nullptr;
bool g_minhookInitialized = false;
std::atomic<std::uint64_t> g_hookCallbacks{0};
std::atomic<std::uint64_t> g_candidatesMatched{0};
std::atomic<std::uint64_t> g_layoutRejected{0};
std::atomic<std::uint64_t> g_wheelGuardRejected{0};
std::atomic<std::uint64_t> g_applyCalls{0};
// Debug bins record the normalized inputs that reached the game function.
// These are NOT actual measured wheel angles and must never be labelled so.
std::atomic<std::uint64_t> g_appliedZero{0};
std::atomic<std::uint64_t> g_appliedLow{0};
std::atomic<std::uint64_t> g_appliedMedium{0};
std::atomic<std::uint64_t> g_appliedHigh{0};
// Original.20: strictly observational counts of signed values forwarded to the
// EXISTING ATS wheel function. No new memory writes or hook offsets.
std::atomic<std::uint64_t> g_appliedNegative{0};
std::atomic<std::uint64_t> g_appliedPositive{0};
std::atomic<int> g_lastAppliedMilli{0};
std::atomic<std::uint64_t> g_eventFrames{0};
// Original21: read-only telemetry evidence for multiplayer research.
// A nonzero game-time offset is evidence; ZERO does NOT prove Solo.
constexpr wchar_t kConvoyProbeName[] = L"Local\\EssieuPilotConvoyProbeV1";
constexpr std::uint32_t kConvoyMagic = 0x45504331; // EPC1
#pragma pack(push,1)
struct ConvoyProbePacket {
    std::uint32_t magic, version, seq_begin, flags;
    std::int32_t offset;
    std::uint64_t tick_ms;
    std::uint32_t seq_end;
};
#pragma pack(pop)
static_assert(sizeof(ConvoyProbePacket)==32,"Convoy probe ABI mismatch");
HANDLE g_convoyMap = nullptr;
ConvoyProbePacket* g_convoyView = nullptr;
std::atomic<std::uint32_t> g_convoySequence{0};
std::atomic<bool> g_multiplayerChannelSeen{false};
std::atomic<std::int32_t> g_lastMultiplayerOffset{0};

void publishConvoyProbe() {
    if (!g_convoyView) return;
    const auto seq = g_convoySequence.fetch_add(2, std::memory_order_relaxed) + 2;
    g_convoyView->seq_begin = seq | 1u;
    MemoryBarrier();
    g_convoyView->magic = kConvoyMagic;
    g_convoyView->version = 1;
    const auto multiplayer = g_possibleConvoy.load(std::memory_order_relaxed);
    g_convoyView->flags =
        (g_multiplayerChannelSeen.load(std::memory_order_relaxed) ? 1u : 0u) |
        (multiplayer ? 2u : 0u) | (multiplayer ? 4u : 0u);
    g_convoyView->offset = g_lastMultiplayerOffset.load(std::memory_order_relaxed);
    g_convoyView->tick_ms = GetTickCount64();
    MemoryBarrier();
    g_convoyView->seq_end = seq;
    MemoryBarrier();
    g_convoyView->seq_begin = seq;
}


void say(const scs_log_type_t type, const char* msg)
{
    if (g_log) g_log(type, msg);
}

bool isPageAccessible(const void* pointer, const std::size_t size, const bool write)
{
    if (pointer == nullptr || size == 0) return false;
    const auto addr = reinterpret_cast<std::uintptr_t>(pointer);
    if (addr > std::numeric_limits<std::uintptr_t>::max() - size) return false;
    MEMORY_BASIC_INFORMATION mbi{};
    if (VirtualQuery(pointer, &mbi, sizeof(mbi)) != sizeof(mbi) ||
        mbi.State != MEM_COMMIT || (mbi.Protect & (PAGE_GUARD | PAGE_NOACCESS)))
        return false;
    const auto region = reinterpret_cast<std::uintptr_t>(mbi.BaseAddress);
    if (addr < region || addr + size > region + mbi.RegionSize) return false;
    const DWORD mask = mbi.Protect & 0xff;
    if (write)
        return mask == PAGE_READWRITE || mask == PAGE_EXECUTE_READWRITE ||
               mask == PAGE_WRITECOPY || mask == PAGE_EXECUTE_WRITECOPY;
    return mask == PAGE_READONLY || mask == PAGE_READWRITE ||
           mask == PAGE_EXECUTE_READ || mask == PAGE_EXECUTE_READWRITE ||
           mask == PAGE_WRITECOPY || mask == PAGE_EXECUTE_WRITECOPY;
}

struct CandidateLayout {
    std::size_t angleOffset;
    std::size_t wheelPtrOffset;
};

// Only explicitly recognized trailer vtables from this EXACT executable.
// Never substitute a guess for an unknown vtable or blindly scan memory.
bool candidateTrailerObject(void* ptr, CandidateLayout& out)
{
    if (!isPageAccessible(ptr, kLegacyWheelPtrOffset + sizeof(void*), false))
        return false;
    const auto vtable = *static_cast<const std::uintptr_t*>(ptr);
    bool known = false;
    for (std::uintptr_t expected : kVirtualTableRvas)
        if (vtable == g_imageBase + expected) known = true;
    if (!known) return false;

    const auto advance = g_imageBase + kSteeringAdvanceRva;
    // TS Extra Utilities' independently documented slot 73 is checked
    // before the older speculative slot 78. A slot must point to the exact
    // same EXE function which MinHook intercepts.
    if (isPageAccessible(reinterpret_cast<const void*>(vtable + kSourceSlot * 8),
                         sizeof(std::uintptr_t), false) &&
        *reinterpret_cast<const std::uintptr_t*>(vtable + kSourceSlot * 8) == advance)
    {
        out = {kSourceAngleOffset, kSourceWheelPtrOffset};
        return true;
    }
    if (isPageAccessible(reinterpret_cast<const void*>(vtable + kLegacySlot * 8),
                         sizeof(std::uintptr_t), false) &&
        *reinterpret_cast<const std::uintptr_t*>(vtable + kLegacySlot * 8) == advance)
    {
        out = {kLegacyAngleOffset, kLegacyWheelPtrOffset};
        return true;
    }
    return false;
}

// This is a **candidate** hook. The call signature, vtable dynamic type,
// and group-of-axles semantics remain to be confirmed by a solo game test.
std::uint64_t __fastcall steeringAdvance(void* self)
{
    const auto original = g_originalAdvance;
    if (!original) return 0;
    g_hookCallbacks.fetch_add(1, std::memory_order_relaxed);
    // Preserve the original ATS steering state before the experimental
    // override. Never skip ATS physics or manipulate an unknown class.
    const std::uint64_t result = original(self);
    if (!g_hookEnabled.load(std::memory_order_relaxed) ||
        g_shuttingDown.load(std::memory_order_relaxed) ||
        g_paused.load(std::memory_order_relaxed) ||
        g_possibleConvoy.load(std::memory_order_relaxed) || !g_applyWheels)
        return result;

    epbridge::Target target{};
    if (!epbridge::read(0, target) ||
        !std::isfinite(target.angle) ||
        std::fabs(target.angle) > 1.0f) return result;

    CandidateLayout layout{};
    if (!candidateTrailerObject(self, layout))
    {
        g_layoutRejected.fetch_add(1, std::memory_order_relaxed);
        return result;
    }
    g_candidatesMatched.fetch_add(1, std::memory_order_relaxed);
    auto* raw = static_cast<unsigned char*>(self);
    if (!isPageAccessible(raw + layout.wheelPtrOffset,sizeof(void*),false) ||
        !isPageAccessible(raw + layout.angleOffset,sizeof(float),true))
    {
        g_wheelGuardRejected.fetch_add(1, std::memory_order_relaxed);
        return result;
    }
    void* wheelData = nullptr;
    std::memcpy(&wheelData, raw + layout.wheelPtrOffset, sizeof(wheelData));
    if (!isPageAccessible(wheelData, 0x60, false))
    {
        g_wheelGuardRejected.fetch_add(1, std::memory_order_relaxed);
        return result;
    }
    // Stop on implausible current steering state instead of overwriting a
    // field which might be something else on this exact game build.
    float previous = 0;
    std::memcpy(&previous,raw + layout.angleOffset,sizeof(previous));
    if (!std::isfinite(previous) || std::fabs(previous)>1.01f)
    {
        g_wheelGuardRejected.fetch_add(1, std::memory_order_relaxed);
        return result;
    }

    std::memcpy(raw + layout.angleOffset, &target.angle, sizeof(target.angle));
    g_applyWheels(wheelData, target.angle);
    if (target.angle < -0.0001f) g_appliedNegative.fetch_add(1, std::memory_order_relaxed);
    if (target.angle > 0.0001f) g_appliedPositive.fetch_add(1, std::memory_order_relaxed);
    // The DLL must reveal whether ATS really RECEIVED distinct magnitudes.
    // Seeing the same wheel angle at 0.63 vs 1.0 cannot be diagnosed from
    // IPC alone; these counters are native-call evidence only.
    const float absolute = std::fabs(target.angle);
    if (absolute < 0.0001f) g_appliedZero.fetch_add(1, std::memory_order_relaxed);
    else if (absolute < 0.25f) g_appliedLow.fetch_add(1, std::memory_order_relaxed);
    else if (absolute < 0.75f) g_appliedMedium.fetch_add(1, std::memory_order_relaxed);
    else g_appliedHigh.fetch_add(1, std::memory_order_relaxed);
    g_lastAppliedMilli.store(static_cast<int>(target.angle*1000.0f), std::memory_order_relaxed);
    g_applyCalls.fetch_add(1, std::memory_order_relaxed);
    epbridge::acknowledgeApplied(0); // Callback only, not wheel proof.
    return result;
}

void SCSAPIFUNC onFrame(const scs_event_t event, const void* const,
                       const scs_context_t)
{
    if (event == SCS_TELEMETRY_EVENT_paused)
        g_paused.store(true, std::memory_order_relaxed);
    if (event == SCS_TELEMETRY_EVENT_started)
        g_paused.store(false, std::memory_order_relaxed);
    if (event != SCS_TELEMETRY_EVENT_frame_end ||
        g_shuttingDown.load(std::memory_order_relaxed)) return;
    publishConvoyProbe();

    // Log counters from the official SDK callback, never from the unverified
    // physics hook's possibly different engine thread. Every ~300 frames.
    if (g_hookEnabled.load(std::memory_order_relaxed))
    {
        const auto frame = g_eventFrames.fetch_add(1, std::memory_order_relaxed);
        if (frame == 300 || (frame > 300 && frame % 900 == 0))
        {
            char msg[240]{};
            std::snprintf(msg,sizeof(msg),
                "EssieuPilot R&D hook counts: calls=%llu candidates=%llu layout_reject=%llu memory_reject=%llu applied=%llu (NOT wheel proof)",
                static_cast<unsigned long long>(g_hookCallbacks.load()),
                static_cast<unsigned long long>(g_candidatesMatched.load()),
                static_cast<unsigned long long>(g_layoutRejected.load()),
                static_cast<unsigned long long>(g_wheelGuardRejected.load()),
                static_cast<unsigned long long>(g_applyCalls.load()));
            say(SCS_LOG_TYPE_message,msg);
            char angles[245]{};
            std::snprintf(angles,sizeof(angles),
                "EssieuPilot R&D input bins (NOT measured wheel degrees): zero=%llu low=%llu medium=%llu high=%llu last_milli=%d",
                static_cast<unsigned long long>(g_appliedZero.load()),
                static_cast<unsigned long long>(g_appliedLow.load()),
                static_cast<unsigned long long>(g_appliedMedium.load()),
                static_cast<unsigned long long>(g_appliedHigh.load()),
                g_lastAppliedMilli.load());
            say(SCS_LOG_TYPE_message,angles);
            char signedCounts[250]{};
            std::snprintf(signedCounts,sizeof(signedCounts),
                "EssieuPilot R&D signed callbacks (NOT wheel motion): negative=%llu positive=%llu last_milli=%d",
                static_cast<unsigned long long>(g_appliedNegative.load()),
                static_cast<unsigned long long>(g_appliedPositive.load()),
                g_lastAppliedMilli.load());
            say(SCS_LOG_TYPE_message,signedCounts);
        }
        return;
    }
    if (g_paused.load(std::memory_order_relaxed) ||
        g_possibleConvoy.load(std::memory_order_relaxed) ||
        !g_hookAddress || !g_originalAdvance) return;

    epbridge::Target requested{};
    // Only arm the hook when EssieuPilot sent a fresh enabled packet after
    // explicit user consent. A normal app launch sends disabled packets.
    if (!epbridge::read(0, requested)) return;
    if (MH_EnableHook(g_hookAddress) == MH_OK)
    {
        g_hookEnabled.store(true, std::memory_order_relaxed);
        say(SCS_LOG_TYPE_warning,
            "EssieuPilot: R&D physics hook armed on this exact ATS executable. SOLO TEST ONLY.");
    }
    else say(SCS_LOG_TYPE_error, "EssieuPilot: native hook enable failed.");
}

// Convoi's time offset can be nonzero, but zero is NOT proof of solo mode.
// This guard stops intervention if the game explicitly reports a remote
// multiplayer time offset. Do not claim full Convoi detection or sync.
void SCSAPIFUNC onMultiplayer(const scs_string_t, const scs_u32_t,
                             const scs_value_t* const value, const scs_context_t)
{
    if (value && value->type == SCS_VALUE_TYPE_s32) {
        const auto offset = value->value_s32.value;
        g_multiplayerChannelSeen.store(true, std::memory_order_relaxed);
        g_lastMultiplayerOffset.store(offset, std::memory_order_relaxed);
        // Once detected, this session stays locked for the safety of other
        // players' trailer objects. Restart ATS before returning to Solo.
        if (offset != 0 && !g_possibleConvoy.exchange(true, std::memory_order_relaxed)) {
            say(SCS_LOG_TYPE_warning,
                "EssieuPilot Original21: multiplayer time offset seen; physical steering suspended until ATS restart. Remote trailer ownership is not validated.");
        }
        publishConvoyProbe();
    }
}

bool checkRuntimeRvas()
{
    if (!g_imageBase) return false;
    auto* signature = reinterpret_cast<const std::uint8_t*>(
        g_imageBase + kApplyWheelsRva);
    if (!isPageAccessible(signature, ep_ats162::STEERING_SIGNATURE.size(), false))
        return false;
    for (std::size_t i = 0; i < ep_ats162::STEERING_SIGNATURE.size(); ++i)
        if (ep_ats162::STEERING_SIGNATURE[i] >= 0 &&
            signature[i] != ep_ats162::STEERING_SIGNATURE[i]) return false;
    g_applyWheels = reinterpret_cast<ApplyWheelsFn>(
        g_imageBase + kApplyWheelsRva);
    g_hookAddress = reinterpret_cast<void*>(g_imageBase + kSteeringAdvanceRva);
    return isPageAccessible(g_hookAddress, 16, false);
}
}

extern "C" __declspec(dllexport)
SCSAPI_RESULT scs_telemetry_init(const scs_u32_t version,
                                 const scs_telemetry_init_params_t* const params)
{
    if (!params || (version != SCS_TELEMETRY_VERSION_1_00 &&
                    version != SCS_TELEMETRY_VERSION_1_01))
        return SCS_RESULT_unsupported;

    const auto* init = static_cast<const scs_telemetry_init_params_v100_t*>(params);
    g_log = init->common.log;
    // CRITICAL: common.game_version is the ATS *telemetry schema* version
    // (1.8 in ATS 1.62.2.2), NOT the displayed game version (1.62).
    // Original.10 mistakenly compared the schema minor to 62 and refused
    // the supported SCS API, producing the user's "does not support any
    // available version of the telemetry api" error.
    if (!init->common.game_id ||
        std::strcmp(init->common.game_id, SCS_GAME_ID_ATS) != 0)
    {
        say(SCS_LOG_TYPE_error, "EssieuPilot: game_id is not ATS; refusing plugin initialization.");
        return SCS_RESULT_unsupported;
    }
    const auto telemetrySchemaMajor = SCS_GET_MAJOR_VERSION(init->common.game_version);
    const auto telemetrySchemaMinor = SCS_GET_MINOR_VERSION(init->common.game_version);
    if (telemetrySchemaMajor != 1)
    {
        say(SCS_LOG_TYPE_error,
            "EssieuPilot: unsupported major ATS telemetry schema (expected major 1).");
        return SCS_RESULT_unsupported;
    }
    char schemaLog[160]{};
    std::snprintf(schemaLog, sizeof(schemaLog),
        "EssieuPilot: SDK API %u.%u, ATS telemetry schema %u.%u (NOT ATS game version).",
        static_cast<unsigned>(SCS_GET_MAJOR_VERSION(version)),
        static_cast<unsigned>(SCS_GET_MINOR_VERSION(version)),
        static_cast<unsigned>(telemetrySchemaMajor),
        static_cast<unsigned>(telemetrySchemaMinor));
    say(SCS_LOG_TYPE_message, schemaLog);

    // Version 1.62.2.2 of the *actual executable* is independently checked
    // using its exact SHA-256, size and function signature below.
    if (!ep_ats162::exactKnownBuild())
    {
        say(SCS_LOG_TYPE_warning,
            "EssieuPilot: ATS binary not exactly 1.62.2.2 known SHA-256; physics plugin disabled.");
        return SCS_RESULT_unsupported;
    }
    g_imageBase = reinterpret_cast<std::uintptr_t>(GetModuleHandleW(nullptr));
    if (!checkRuntimeRvas())
    {
        say(SCS_LOG_TYPE_error,
            "EssieuPilot: steering function RVA or in-memory signature mismatch; no native hook.");
        return SCS_RESULT_unsupported;
    }
    g_paused.store(true);
    g_possibleConvoy.store(false);
    g_multiplayerChannelSeen.store(false);
    g_lastMultiplayerOffset.store(0);
    g_convoySequence.store(0);
    g_shuttingDown.store(false);
    g_hookEnabled.store(false);
    g_hookCallbacks.store(0);
    g_candidatesMatched.store(0);
    g_layoutRejected.store(0);
    g_wheelGuardRejected.store(0);
    g_applyCalls.store(0);
    g_appliedZero.store(0);
    g_appliedLow.store(0);
    g_appliedMedium.store(0);
    g_appliedHigh.store(0);
    g_lastAppliedMilli.store(0);
    g_eventFrames.store(0);

    if (!init->register_for_event ||
        init->register_for_event(SCS_TELEMETRY_EVENT_paused, &onFrame, nullptr) != SCS_RESULT_ok ||
        init->register_for_event(SCS_TELEMETRY_EVENT_started, &onFrame, nullptr) != SCS_RESULT_ok ||
        init->register_for_event(SCS_TELEMETRY_EVENT_frame_end, &onFrame, nullptr) != SCS_RESULT_ok)
        return SCS_RESULT_generic_error;

    // Optional observer: the SDK may not support this channel. It cannot
    // guarantee whether the user is in Convoy; user must test SOLO only.
    if (init->register_for_channel)
        (void)init->register_for_channel("multiplayer.time.offset", SCS_U32_NIL,
             SCS_VALUE_TYPE_s32, SCS_TELEMETRY_CHANNEL_FLAG_each_frame,
             &onMultiplayer, nullptr);

    if (MH_Initialize() != MH_OK)
    {
        say(SCS_LOG_TYPE_error, "EssieuPilot: MinHook initialization failed.");
        return SCS_RESULT_generic_error;
    }
    g_minhookInitialized = true;
    if (MH_CreateHook(g_hookAddress, reinterpret_cast<void*>(&steeringAdvance),
                      reinterpret_cast<void**>(&g_originalAdvance)) != MH_OK ||
        !g_originalAdvance)
    {
        say(SCS_LOG_TYPE_error,
            "EssieuPilot: steering candidate hook preparation failed.");
        MH_Uninitialize();
        g_minhookInitialized = false;
        return SCS_RESULT_generic_error;
    }
    g_convoyMap = CreateFileMappingW(INVALID_HANDLE_VALUE, nullptr, PAGE_READWRITE,
        0, sizeof(ConvoyProbePacket), kConvoyProbeName);
    if (g_convoyMap) {
        g_convoyView = static_cast<ConvoyProbePacket*>(
            MapViewOfFile(g_convoyMap, FILE_MAP_WRITE, 0, 0, sizeof(ConvoyProbePacket)));
        if (!g_convoyView) { CloseHandle(g_convoyMap); g_convoyMap = nullptr; }
    }
    publishConvoyProbe();
    epbridge::announceSdkReady(1, 62);
    say(SCS_LOG_TYPE_warning,
        "EssieuPilot: experimental 1.62.2.2 DLL loaded; physics inactive until explicit test consent + fresh IPC.");
    return SCS_RESULT_ok;
}

extern "C" __declspec(dllexport)
SCSAPI_VOID scs_telemetry_shutdown(void)
{
    g_shuttingDown.store(true);
    g_hookEnabled.store(false);
    if (g_minhookInitialized)
    {
        if (g_hookAddress) (void)MH_DisableHook(g_hookAddress);
        if (g_hookAddress) (void)MH_RemoveHook(g_hookAddress);
        (void)MH_Uninitialize();
        g_minhookInitialized = false;
    }
    if (g_convoyView) { UnmapViewOfFile(g_convoyView); g_convoyView = nullptr; }
    if (g_convoyMap) { CloseHandle(g_convoyMap); g_convoyMap = nullptr; }
    g_originalAdvance = nullptr;
    g_applyWheels = nullptr;
    g_hookAddress = nullptr;
}

BOOL WINAPI DllMain(HMODULE, DWORD, LPVOID) { return TRUE; }
