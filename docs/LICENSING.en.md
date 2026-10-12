# Project licensing

[简体中文](LICENSING.md)

## Original work: MIT

The original Go application code, documentation, and original logo use the MIT
License in the root LICENSE file. The notice names "qqmusic-convert contributors";
each contributor retains the rights they actually hold. It does not claim ownership
of third-party work or authorize material a contributor cannot license.

MIT permits use, modification, copying, distribution, and commercial use, subject
to preserving its copyright and license notice. It includes a warranty disclaimer.
The English LICENSE is the operative text; this explanation adds no noncommercial
or other restrictions to it.

The software license grants no rights to music recordings. The intended use with
local audio a user is entitled to access is not an extra restriction on the MIT license.

## Third-party work retains its own terms

| Work | License | Retained text |
|---|---|---|
| QM Unlock-derived QMC2 / TEA and algorithm vectors | MIT, with original attribution | licenses/QMUnlock-MIT.txt |
| Go runtime and standard library | BSD | licenses/Go-BSD.txt |
| golang.org/x/sys v0.49.0 | BSD | licenses/xsys-BSD.txt |
| rsrc v0.10.2 build tool | MIT | licenses/rsrc-MIT.txt |
| Current bundled FFmpeg / FFprobe | Publisher states GPLv3; linked libraries also require review | licenses/FFmpeg-GPL.txt |

NOTICE.md records sources, versions, copyrights, and component scope. The root MIT
license does not relicense FFmpeg or any other third-party work. Keeping a GPL text
does not by itself establish that every distribution requirement has been met.

FFmpeg runs as a separate process, while its compressed binaries are embedded in
the EXE. This records the implementation, not a blanket legal determination that
all combinations are aggregates or that all original application code automatically
becomes GPL-licensed.

## Current public-release boundary

- Original source and MIT texts are prepared. Preserve third-party notices when sharing source.
- The current executable's materials form a local preview, not a public-binary-release clearance.
- The publisher-declared FFmpeg source revision has been obtained. It does not cover
  all sources for the statically linked dependency set.
- Exact dependency sources, patches, build scripts, toolchain records, and notices
  still require verification.

See [EXE distribution materials](DISTRIBUTION.md) for concrete outstanding items.
Do not publish an unfulfilled source offer or describe the package as fully license-compliant.

## Official references

- MIT text: https://opensource.org/license/mit
- FFmpeg guidance: https://ffmpeg.org/legal.html
- GPLv3 text: https://www.gnu.org/licenses/gpl-3.0.html

Requirements depend on the actual licenses and distributed combination. Obtain a
qualified review where the interpretation of that combination is uncertain.
