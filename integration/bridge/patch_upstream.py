"""Research-only adaptation of MIT TS Extra Utilities at pinned commit.
Never ship this DLL until ATS 1.62 layout verification and in-game testing.
"""
from pathlib import Path
import shutil,sys
root=Path(sys.argv[1]).resolve()
here=Path(__file__).resolve().parent
src=root/'src/windows/trailer_manipulation.cpp'
head=root/'src/windows/trailer_manipulation.hpp'
core=root/'src/core.cpp'
game=root/'src/main.cpp'
assert all(p.is_file() for p in [src,head,core,game])
shutil.copyfile(here/'EssieuPilotBridge.hpp',root/'src/EssieuPilotBridge.hpp')
def change(s,a,b):
    if s.count(a)!=1: raise ValueError(f"Expected once, found {s.count(a)}: {a[:70]!r}")
    return s.replace(a,b,1)
s=src.read_text(encoding='utf-8-sig')
s=change(s,'#include "imgui.h"','#include "imgui.h"\n#include "EssieuPilotBridge.hpp"\n#include <cmath>')
s=change(s,'    thread_local std::shared_ptr< CVirtualFunctionHook > steering_advance_hook = nullptr;',
  '    std::shared_ptr< CVirtualFunctionHook > steering_advance_hook = nullptr;\n    prism::set_individual_steering_fn* ep_set_steering = nullptr;')
s=change(s,'while ( check_trailer != nullptr && self != check_trailer )','while ( check_trailer != nullptr && self != check_trailer && trailer_index < 20 )')
s=change(s,
'''        if ( check_trailer == nullptr || locked_trailers[ trailer_index ] == false )
        {
            return steering_advance_hook->get_original< prism::physics_trailer_u_steering_advance_fn >()( self );
        }
        return 0;''',
'''        if ( steering_advance_hook == nullptr ) return 0;
        const auto original=steering_advance_hook->get_original< prism::physics_trailer_u_steering_advance_fn >();
        if (check_trailer == nullptr || trailer_index < 0 || trailer_index >= 20) return original(self);
        epbridge::Target target{};
        if (ep_set_steering != nullptr && self->wheel_steering_stuff != nullptr &&
            epbridge::read(static_cast<uint32_t>(trailer_index), target)) {
            locked_trailers[trailer_index]=true;
            self->steering=target.angle;
            ep_set_steering(self->wheel_steering_stuff,target.angle);
            epbridge::acknowledgeApplied(static_cast<uint32_t>(trailer_index));
            return 0;
        }
        // Missing/stale IPC = immediately restore the game's automatic steering.
        locked_trailers[trailer_index]=false;
        return original(self);''')
s=change(s,'        if ( this->set_individual_steering_fn_ == nullptr )',
         '        ep_set_steering = this->set_individual_steering_fn_;\n        if ( this->set_individual_steering_fn_ == nullptr )')
s=change(s,
'''        if (
            CCore::g_instance->get_game_actor() == nullptr ||''',
'''        if ( !valid_ || CCore::g_instance == nullptr ||
            CCore::g_instance->get_game_actor() == nullptr ||''')
marker='    void CTrailerManipulation::render_trailers() const\n    {'
start=s.index('        if ( steering_advance_hook == nullptr )',s.index(marker))
end=s.index('        int i = 0;',start)
s=s[:start]+'        this->ensure_bridge_hook();\n\n'+s[end:]
s=change(s,'''        do
        {
            const auto trailer_name''','''        do
        {
            if (i >= 20) break;
            const auto trailer_name''')
method='''    // Called every graphical frame, whether or not the UI overlay is visible.
    void CTrailerManipulation::ensure_bridge_hook() const
    {
        if (!valid_ || steering_advance_hook != nullptr || CCore::g_instance == nullptr) return;
        const auto* actor=CCore::g_instance->get_game_actor();
        if (actor == nullptr || actor->game_trailer_actor == nullptr ||
            this->set_individual_steering_fn_ == nullptr) return;
        auto* trailer=actor->game_trailer_actor;
        const auto vtable=*reinterpret_cast< uint64_t** >(trailer);
        if (vtable == nullptr) return;
        const auto address=reinterpret_cast<uint64_t>(vtable + 73);
        steering_advance_hook=CCore::g_instance->get_hooks_manager()->register_virtual_function_hook(
            "physics_trailer_u::steering_advance",address,
            reinterpret_cast<uint64_t>(&hk_steering_advance));
        if (steering_advance_hook && steering_advance_hook->hook()!=CHook::HOOKED)
            CCore::g_instance->error("Could not install EssieuPilot steering hook");
    }

'''
s=change(s,marker,method+marker)
# During this first ATS test, absolutely DO NOT enable TS Extra's unrelated
# trailer disconnect or auto-reconnect hooks: their old signatures are unverified
# on ATS 1.62 and create avoidable crash risks.
hook_start=s.index('        const auto crash_fn_address = memory::get_address_for_pattern(')
hook_end=s.index('        this->valid_ = true;',hook_start)
s=s[:hook_start]+'''        // EssieuPilot ATS test: legacy disconnect/joint hooks removed.
        // They are not related to steering and their memory patterns
        // have not been validated on ATS 1.62.

'''+s[hook_end:]
# Delay the high-risk physics vtable hook until the user has actively enabled
# steering via a fresh IPC command, not just when the trailer renders.
s=change(s,
   '        if (!valid_ || steering_advance_hook != nullptr || CCore::g_instance == nullptr) return;',
   '''        if (!valid_ || steering_advance_hook != nullptr || CCore::g_instance == nullptr) return;
        epbridge::Target preflight{};
        if (!epbridge::read(0,preflight)) return;''')
# Hide unsupported trailer joint/disconnect buttons completely in the test UI.
s=change(s,
   '        this->render_trailers();',
   '        ImGui::TextWrapped("EssieuPilot ATS 1.62: experimental physical steering only. Joint manipulation disabled.");')
src.write_text(s,encoding='utf-8')
h=head.read_text(encoding='utf-8-sig')
h=change(h,'        void render() override;', '        void render() override;\n        void ensure_bridge_hook() const;')
head.write_text(h,encoding='utf-8')
c=core.read_text(encoding='utf-8-sig')
c=change(c,'    CCore* CCore::g_instance = nullptr;',
         '    CCore* CCore::g_instance = nullptr;\n    static std::weak_ptr<CTrailerManipulation> ep_tracker;')
c=change(c,'        const auto trailer_manipulation = this->window_manager_->register_window( std::make_shared< CTrailerManipulation >() );',
         '        const auto trailer_manipulation = this->window_manager_->register_window( std::make_shared< CTrailerManipulation >() );\n        ep_tracker=trailer_manipulation;')
c=change(c,'        ImGui::NewFrame();',
         '        ImGui::NewFrame();\n        if (auto tracker=ep_tracker.lock()) tracker->ensure_bridge_hook();')
core.write_text(c,encoding='utf-8')
m=game.read_text(encoding='utf-8-sig')
m=change(m,'#include "scssdk.h"', '#include "scssdk.h"\n#include <cstring>\n#include <amtrucks/scssdk_ats.h>')
m=change(m,'    auto* init_params = static_cast< const scs_telemetry_init_params_v100_t* >( params );',
'''    if (params == nullptr || (version != SCS_TELEMETRY_VERSION_1_01 && version != SCS_TELEMETRY_VERSION_1_00))
        return SCS_RESULT_unsupported;
    auto* init_params = static_cast< const scs_telemetry_init_params_v100_t* >( params );
    // Fail closed on any other game or unknown minor version.
    if (init_params->common.game_id == nullptr ||
        strcmp(init_params->common.game_id,SCS_GAME_ID_ATS)!=0 ||
        SCS_GET_MAJOR_VERSION(init_params->common.game_version)!=1 ||
        SCS_GET_MINOR_VERSION(init_params->common.game_version)!=62)
        return SCS_RESULT_unsupported;
    HANDLE bridge=OpenFileMappingW(FILE_MAP_READ,FALSE,L"Local\\EssieuPilotSteeringV1");
    if(!bridge)return SCS_RESULT_unsupported;
    CloseHandle(bridge);''')
game.write_text(m,encoding='utf-8')
print('R&D bridge patched; no install or release permitted without ATS 1.62 verification.')
