# Source availability — local release preview

**Public binary release status: NOT_READY_FOR_PUBLIC_BINARY_RELEASE**

The original qqmusic-convert source is prepared in the accompanying application
source archive. It includes the Go implementation, public synthetic test vectors,
build scripts, MIT LICENSE, NOTICE, and retained third-party notices. It contains
no real audio, login tokens, song keys, or embedded third-party executable archives.

Application source repository: https://github.com/navy6688/qqmusic-convert
This repository covers the application source; it does not supply the missing
Corresponding Source for the complete bundled static FFmpeg build.

The publisher-declared upstream FFmpeg source was obtained at:
239f2c733de417201d7ad3b3b8b0d9b63285b2b1.
The archive includes FFmpeg's own configure script, release metadata, and GPL text.
This is ONLY the FFmpeg upstream source, not the full Corresponding Source for the
publisher's statically linked FFmpeg/FFprobe executables.

Still unresolved:
- Exact source revisions and archives for every linked external dependency.
- Dependencies with unspecified versions, including openal-soft recorded as "latest".
- Transitive dependency and runtime-library scope.
- Publisher patches and complete dependency/build/link/install scripts.
- A sufficient toolchain/build environment record and dependency license notices.
- An actual, maintained public release location for complete corresponding materials.

The application source repository is listed above. No binary release page or
written source-offer commitment is claimed. This preview is not a claim that GPL
distribution conditions are met.

For a future network binary release, place clear version-specific source-access
instructions alongside the executable and implement the applicable GPLv3 section
6 option. Publishing a homepage URL or one upstream archive alone does not
establish completeness. Do not publish an unfulfilled written source offer.

See docs/DISTRIBUTION.md, docs/LICENSING.md, NOTICE.md, and the retained GPL text.
