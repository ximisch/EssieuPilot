#pragma once
// EssieuPilot physical research only. Static executable provenance gate.
// This is NOT a validation of game offsets, vtable entries, physics or wheel animation.
// No hook must be installed unless all later ABI validations also pass.
#define WIN32_LEAN_AND_MEAN
#include <Windows.h>
#include <bcrypt.h>
#include <array>
#include <cstdint>
#include <cstring>
#include <cwchar>
#include <vector>

#pragma comment(lib, "bcrypt.lib")

namespace ep_ats162
{
  constexpr std::uint64_t EXPECTED_SIZE = 53640592ULL;
  constexpr std::size_t STEERING_SIGNATURE_FILE_OFFSET = 0x84CD40;
  constexpr std::array<std::uint8_t, 32> EXPECTED_SHA256 = {
    0x63,0xec,0xae,0x85,0xa0,0x70,0x93,0x45,
    0x30,0xd9,0xfd,0x97,0x1d,0x4d,0x56,0xe5,
    0x47,0x9f,0x12,0x14,0xd7,0x07,0x7b,0x40,
    0x3f,0x79,0xb8,0x98,0x1c,0x61,0x3e,0x08
  };
  // The wildcards are unknown bytes of the old TS Extra steering pattern.
  // A match is only a weak positive signal: ABI still not validated.
  constexpr std::array<int, 24> STEERING_SIGNATURE = {
    0x48,0x89,0x5c,0x24,0x08,0x48,0x89,0x74,
    0x24,0x10,0x57,0x48,0x83,0xec,-1,0x8b,
    0x41,-1,0x48,0x8b,0xd9,0x0f,0x29,0x74
  };

  struct UniqueFile
  {
    HANDLE value = INVALID_HANDLE_VALUE;
    ~UniqueFile() { if (value != INVALID_HANDLE_VALUE) CloseHandle(value); }
  };

  struct UniqueAlgorithm
  {
    BCRYPT_ALG_HANDLE value = nullptr;
    ~UniqueAlgorithm() { if (value != nullptr) BCryptCloseAlgorithmProvider(value, 0); }
  };

  struct UniqueHash
  {
    BCRYPT_HASH_HANDLE value = nullptr;
    ~UniqueHash() { if (value != nullptr) BCryptDestroyHash(value); }
  };

  inline bool expectedSignatureAt(const std::vector<unsigned char>& image) noexcept
  {
    if (STEERING_SIGNATURE_FILE_OFFSET + STEERING_SIGNATURE.size() > image.size())
      return false;
    for (std::size_t i = 0; i < STEERING_SIGNATURE.size(); ++i)
      if (STEERING_SIGNATURE[i] >= 0 &&
          image[STEERING_SIGNATURE_FILE_OFFSET + i] != STEERING_SIGNATURE[i])
        return false;
    return true;
  }

  // Reads the currently executing game EXE from disk; it NEVER writes to,
  // injects into, or scans the in-memory game process.
  inline bool exactKnownBuild()
  {
    std::array<wchar_t, 32768> path{};
    DWORD count = GetModuleFileNameW(nullptr, path.data(), static_cast<DWORD>(path.size()));
    if (count == 0 || count >= path.size()) return false;
    const wchar_t* base = wcsrchr(path.data(), L'\\');
    base = base ? base + 1 : path.data();
    if (_wcsicmp(base, L"amtrucks.exe") != 0) return false;

    UniqueFile file;
    file.value = CreateFileW(path.data(), GENERIC_READ,
        FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE, nullptr,
        OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, nullptr);
    if (file.value == INVALID_HANDLE_VALUE) return false;
    LARGE_INTEGER size{};
    if (!GetFileSizeEx(file.value, &size) ||
        size.QuadPart != static_cast<LONGLONG>(EXPECTED_SIZE)) return false;

    std::vector<unsigned char> bytes(static_cast<std::size_t>(EXPECTED_SIZE));
    std::size_t consumed = 0;
    while (consumed < bytes.size())
    {
      DWORD readCount = 0;
      DWORD chunk = static_cast<DWORD>(
          (bytes.size() - consumed) > 1024 * 1024 ? 1024 * 1024 : (bytes.size() - consumed));
      if (!ReadFile(file.value, bytes.data() + consumed, chunk, &readCount, nullptr) ||
          readCount == 0) return false;
      consumed += readCount;
    }
    if (!expectedSignatureAt(bytes)) return false;

    UniqueAlgorithm algo;
    if (BCryptOpenAlgorithmProvider(&algo.value, BCRYPT_SHA256_ALGORITHM, nullptr, 0) < 0)
      return false;
    DWORD objectSize = 0, returned = 0;
    if (BCryptGetProperty(algo.value, BCRYPT_OBJECT_LENGTH,
         reinterpret_cast<PUCHAR>(&objectSize), sizeof(objectSize), &returned, 0) < 0 ||
        objectSize == 0 || objectSize > 4096) return false;

    std::vector<unsigned char> hashBuffer(objectSize);
    UniqueHash hash;
    if (BCryptCreateHash(algo.value, &hash.value, hashBuffer.data(),
         objectSize, nullptr, 0, 0) < 0) return false;

    for (std::size_t index = 0; index < bytes.size(); index += 1024 * 1024)
    {
      ULONG length = static_cast<ULONG>(
          ((bytes.size() - index) < 1024 * 1024) ? bytes.size() - index : 1024 * 1024);
      if (BCryptHashData(hash.value, bytes.data() + index, length, 0) < 0)
        return false;
    }
    std::array<std::uint8_t, 32> digest{};
    if (BCryptFinishHash(hash.value, digest.data(),
        static_cast<ULONG>(digest.size()), 0) < 0) return false;

    return digest == EXPECTED_SHA256;
  }
}
