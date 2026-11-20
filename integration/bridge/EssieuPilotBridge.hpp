#pragma once
// R&D ONLY — physical in-game trailer steering ABI, not ATS 1.62 validated.
#define WIN32_LEAN_AND_MEAN
#include <Windows.h>
#include <cmath>
#include <cstdint>
#include <cstring>
namespace epbridge {
constexpr wchar_t MAP_NAME[] = L"Local\\EssieuPilotSteeringV1";
constexpr uint32_t MAGIC=0x45505331;
constexpr uint32_t VERSION=1;
constexpr uint64_t MAX_AGE_MS=250;
#pragma pack(push,1)
struct Packet {
  uint32_t magic,version,seq_begin,enabled;
  uint64_t tick_ms;
  uint32_t trailer_index;
  float angle_normalized;
  uint32_t seq_end;
};
#pragma pack(pop)
static_assert(sizeof(Packet)==36,"bridge ABI mismatch");
struct Target { bool enabled; float angle; };
inline bool read(uint32_t requested, Target& out) {
  static thread_local HANDLE handle=nullptr;
  static thread_local const void* view=nullptr;
  if (!view) {
    handle=OpenFileMappingW(FILE_MAP_READ,FALSE,MAP_NAME);
    if (!handle) return false;
    view=MapViewOfFile(handle,FILE_MAP_READ,0,0,sizeof(Packet));
    if (!view) { CloseHandle(handle);handle=nullptr;return false; }
  }
  auto* shared=static_cast<const Packet*>(view);
  Packet copy{};
  bool stable=false;
  for(int attempt=0;attempt<2;++attempt) {
    uint32_t seq=shared->seq_begin;
    MemoryBarrier();
    std::memcpy(&copy,view,sizeof copy);
    MemoryBarrier();
    stable=(seq==shared->seq_begin && !(seq&1u) && copy.seq_end==seq);
    if(stable)break;
  }
  if(!stable || copy.magic!=MAGIC || copy.version!=VERSION ||
      copy.enabled!=1 || requested>=20 || copy.trailer_index!=requested ||
      !std::isfinite(copy.angle_normalized) || std::fabs(copy.angle_normalized)>1.f) return false;
  const uint64_t now=GetTickCount64();
  if(now<copy.tick_ms || now-copy.tick_ms>MAX_AGE_MS)return false;
  out={true,copy.angle_normalized};
  return true;
}
}
