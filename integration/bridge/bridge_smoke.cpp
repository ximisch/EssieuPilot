// Bridge contract smoke test; does not connect to or modify any game.
#include "EssieuPilotBridge.hpp"
#include <cstdio>
#include <cstdint>
#include <cstring>
#include <limits>
int main() {
    using namespace epbridge;
    HANDLE h=CreateFileMappingW(INVALID_HANDLE_VALUE,nullptr,PAGE_READWRITE,0,sizeof(Packet),MAP_NAME);
    if(!h) return 10;
    auto* mem=static_cast<Packet*>(MapViewOfFile(h,FILE_MAP_WRITE|FILE_MAP_READ,0,0,sizeof(Packet)));
    if(!mem){CloseHandle(h);return 11;}
    *mem={MAGIC,VERSION,2,1,GetTickCount64(),0,.5f,2};
    Target out{};
    if(!read(0,out)||!out.enabled||out.angle!=.5f) return 12;
    if(read(1,out)) return 13;
    mem->seq_begin=3;
    if(read(0,out)) return 14;
    mem->seq_begin=4;mem->seq_end=4;mem->angle_normalized=1.4f;
    if(read(0,out)) return 15;
    mem->angle_normalized=.5f;mem->tick_ms=GetTickCount64()-1000;
    if(read(0,out)) return 16;
    mem->tick_ms=GetTickCount64();mem->enabled=0;
    if(read(0,out)) return 17;
    UnmapViewOfFile(mem);CloseHandle(h);
    acknowledgeApplied(0);
    HANDLE ackMap=OpenFileMappingW(FILE_MAP_READ,FALSE,ACK_MAP_NAME);
    if(!ackMap) return 18;
    auto* ack=static_cast<const AppliedPacket*>(
        MapViewOfFile(ackMap,FILE_MAP_READ,0,0,sizeof(AppliedPacket)));
    if(!ack) {CloseHandle(ackMap);return 19;}
    if(ack->magic!=ACK_MAGIC || ack->version!=1 || ack->trailer_index!=0 ||
       ack->seq_begin!=ack->seq_end || (ack->seq_begin&1u) ||
       GetTickCount64()<ack->tick_ms ||
       GetTickCount64()-ack->tick_ms>1500) return 20;
    UnmapViewOfFile(ack);CloseHandle(ackMap);
    std::puts("EssieuPilot steering IPC and physical-callback acknowledgement mapping: PASS");
    return 0;
}
