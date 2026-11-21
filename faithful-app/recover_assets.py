"""Recover front-end source and artwork from the user's verified EssieuPilot v1.6.1/v1.6.2.

This script extracts original assets without decompiling proprietary runtime internals.
It refuses unknown binaries; recovered assets are NOT a complete application.
"""
import hashlib
import struct
import sys
from pathlib import Path

HASHES = {
    'bf65000cde9e32f93011ead432a412033015ff1be128444820392eabf4280450',
    'f8bb61b07d83e4f6e9324f07972133a3efacab233b82febd66b9304961f1e29b',
}


def text_between(raw: bytes, prefix: bytes, minimum: int = 100) -> bytes:
    start = raw.find(prefix)
    if start < 0:
        raise ValueError('Missing embedded asset: %r' % prefix[:45])
    end = raw.find(b'\0', start)
    content = raw[start:end]
    if len(content) < minimum:
        raise ValueError('Asset too small: %r' % prefix[:45])
    content.decode('utf-8')
    return content


def png_at(raw: bytes, start: int) -> bytes:
    assert raw[start:start+8] == b'\x89PNG\r\n\x1a\n'
    cursor = start+8
    while cursor+12 <= len(raw):
        n = struct.unpack_from('>I', raw, cursor)[0]
        tag = raw[cursor+4:cursor+8]
        cursor += 12+n
        if tag == b'IEND':
            return raw[start:cursor]
    raise ValueError('PNG without IEND')


def ico_at(raw: bytes, start: int) -> bytes:
    assert raw[start:start+4] == b'\0\0\1\0'
    count = struct.unpack_from('<H', raw, start+4)[0]
    if count < 1 or count > 20:
        raise ValueError('Invalid ICO entries')
    end = max(struct.unpack_from('<II', raw, start+6+16*i+8)[0]+struct.unpack_from('<II', raw, start+6+16*i+8)[1] for i in range(count))
    return raw[start:start+end]


def recover(source: Path, destination: Path) -> None:
    raw = source.read_bytes()
    sha = hashlib.sha256(raw).hexdigest()
    if sha not in HASHES:
        raise ValueError(f'Unrecognized EssieuPilot build, SHA256={sha}')
    destination.mkdir(parents=True, exist_ok=True)
    resources = {
        'index.html': (b'<!doctype html>\n<html lang="fr">', 9000),
        'styles.css': (b':root{color-scheme:dark;--bg:', 10000),
        'modern.css': (b'/* ATS Studio V1.4', 10000),
        'modern.js': (b"'use strict';\n// Interface V1.4.1", 20000),
        'app.js': (b"'use strict';\n\nconst $ = (id)", 25000),
    }
    for name, (prefix, minimum) in resources.items():
        content = text_between(raw, prefix, minimum)
        (destination/name).write_bytes(content)
        print(name, len(content), 'bytes')

    # Steering logic is followed by Go's read-only data, rather than a null terminator.
    start = raw.index(b"'use strict';\n// Moteur independant du jeu")
    terminator = b'  return {analogPower, updateAxle};\n});\n'
    end = raw.index(terminator, start)+len(terminator)
    (destination/'steering_logic.js').write_bytes(raw[start:end])

    # Original interface assets: 5 JPEG feature cards, 1 hero JPEG, PNG logo, Windows ICO.
    jpeg_starts = [i for i in range(3_850_000, 4_010_000) if raw.startswith(b'\xff\xd8\xff', i)]
    if len(jpeg_starts) != 6:
        raise ValueError(f'Expected six JPEG images; got {len(jpeg_starts)}')
    for j, start in enumerate(jpeg_starts):
        end = raw.index(b'\xff\xd9', start+4)+2
        name = f'feature_{j}.jpg' if j < 5 else 'hero_reference.jpg'
        (destination/name).write_bytes(raw[start:end])
    logo = raw.index(b'\x89PNG\r\n\x1a\n', jpeg_starts[-1])
    (destination/'logo.png').write_bytes(png_at(raw, logo))
    icon = raw.index(b'\0\0\1\0', raw.index(b"'use strict';\n\nconst $ = (id)"))
    (destination/'EssieuPilot.ico').write_bytes(ico_at(raw, icon))
    print('Artwork: 6 JPEG, 1 PNG, 1 ICO; all recovered from signed-off known binary hash.')


if __name__ == '__main__':
    if len(sys.argv) != 3:
        raise SystemExit('Usage: recover_assets.py <EssieuPilot.exe> <web-output-dir>')
    recover(Path(sys.argv[1]), Path(sys.argv[2]))
