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
constexpr std::size_t kAngleOffset = 0x6B8;
constexpr std::size_t kWheelDataPtrOffset = 0x6C0;

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

bool candidateTrailerObject(void* ptr)
{
    if (!isPageAccessible(ptr, kWheelDataPtrOffset + sizeof(void*), false))
        return false;
    const auto vtable = *static_cast<const std::uintptr_t*>(ptr);
    bool exact = false;
    for (std::uintptr_t known : kVirtualTableRvas)
        if (vtable == g_imageBase + known) exact = true;
    if (!exact) return false;
    // Independent check: the selected virtual slot of the object MUST be
    // exactly the steering_advance implementation found in the user EXE.
    if (!isPageAccessible(reinterpret_cast<const void*>(vtable + 78 * 8),
                          sizeof(std::uintptr_t), false)) return false;
    return *reinterpret_cast<const std::uintptr_t*>(vtable + 78 * 8) ==
        g_imageBase + kSteeringAdvanceRva;
}

// This is a **candidate** hook. The call signature, vtable dynamic type,
// and group-of-axles semantics remain to be confirmed by a solo game test.
std::uint64_t __fastcall steeringAdvance(void* self)
{
    // Always preserve the game's normal steering logic first. Never retain a
    // previous command after IPC timeout, pause, deactivation or Convoi.
    const auto original = g_originalAdvance;
    if (!original) return 0;
    const std::uint64_t result = original(self);
    if (!g_hookEnabled.load(std::memory_order_relaxed) ||
        g_shuttingDown.load(std::memory_order_relaxed) ||
        g_paused.load(std::memory_order_relaxed) ||
        g_possibleConvoy.load(std::memory_order_relaxed) ||
        !candidateTrailerObject(self) || !g_applyWheels)
        return result;

    epbridge::Target target{};
    if (!epbridge::read(0, target) ||
        !std::isfinite(target.angle) ||
        std::fabs(target.angle) > 1.0f) return result;

    auto* raw = static_cast<unsigned char*>(self);
    void* wheelData = nullptr;
    std::memcpy(&wheelData, raw + kWheelDataPtrOffset, sizeof(wheelData));
    if (!isPageAccessible(wheelData, 0x170, false) ||
        !isPageAccessible(raw + kAngleOffset, sizeof(float), true)) return result;

    // The function discovered in ATS 1.62.2.2 accepts the steering-data
    // pointer and a normalized float. The command can still be ignored
    // by unsteerable axles or superseded by later game physics stages.
    std::memcpy(raw + kAngleOffset, &target.angle, sizeof(target.angle));
    g_applyWheels(wheelData, target.angle);
    epbridge::acknowledgeApplied(0); // Proof of callback only, NOT wheel motion.
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
        g_hookEnabled.load(std::memory_order_relaxed) ||
        g_shuttingDown.load(std::memory_order_relaxed) ||
        g_paused.load(std::memory_order_relaxed) ||
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
    if (value && value->type == SCS_VALUE_TYPE_s32 &&
        value->value_s32.value != 0)
    {
        g_possibleConvoy.store(true, std::memory_order_relaxed);
        say(SCS_LOG_TYPE_warning,
            "EssieuPilot: multiplayer time offset detected; physics output disabled (no Convoy support yet).");
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
    if (!init->common.game_id ||
        std::strcmp(init->common.game_id, SCS_GAME_ID_ATS) != 0 ||
        SCS_GET_MAJOR_VERSION(init->common.game_version) != 1 ||
        SCS_GET_MINOR_VERSION(init->common.game_version) != 62)
        return SCS_RESULT_unsupported;

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
    g_shuttingDown.store(false);
    g_hookEnabled.store(false);

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
    g_originalAdvance = nullptr;
    g_applyWheels = nullptr;
    g_hookAddress = nullptr;
}

BOOL WINAPI DllMain(HMODULE, DWORD, LPVOID) { return TRUE; }
