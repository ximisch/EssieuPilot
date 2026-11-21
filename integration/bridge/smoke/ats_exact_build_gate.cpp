#include "../ATS162ExactBuildGate.hpp"
#include <iostream>
#include <vector>

int main()
{
  // A CI worker is NOT the exact user's ATS build: the gate must refuse.
  if (ep_ats162::exactKnownBuild())
  {
    std::cerr << "Unknown executable was authorized for native steering.\n";
    return 1;
  }

  // A matching signature is necessary but never sufficient for ABI safety.
  std::vector<unsigned char> image(
      ep_ats162::STEERING_SIGNATURE_FILE_OFFSET +
      ep_ats162::STEERING_SIGNATURE.size(), 0);
  for (std::size_t i = 0; i < ep_ats162::STEERING_SIGNATURE.size(); ++i)
  {
    const int expected = ep_ats162::STEERING_SIGNATURE[i];
    if (expected >= 0)
      image[ep_ats162::STEERING_SIGNATURE_FILE_OFFSET + i] =
          static_cast<unsigned char>(expected);
  }
  if (!ep_ats162::expectedSignatureAt(image))
  {
    std::cerr << "Known steering signature rejected.\n";
    return 2;
  }
  image[ep_ats162::STEERING_SIGNATURE_FILE_OFFSET] = 0;
  if (ep_ats162::expectedSignatureAt(image))
  {
    std::cerr << "Corrupted signature was accepted.\n";
    return 3;
  }
  std::cout << "PASS: signature match/rejection, unknown EXE fail-closed; "
               "no ATS memory accessed, no hook installed.\n";
  return 0;
}
