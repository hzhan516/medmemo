# Third-Party Licenses

> 🌐 [中文版本](./docs/i18n/zh-Hans-CN/THIRD_PARTY_LICENSES.md)

> **Project**: MedMemo — Open-source desktop health information tool
> **Project License**: [MIT License](./LICENSE)
> **Last updated**: 2026-07-14

---

## Overview

MedMemo is licensed under the MIT License. This document lists all third-party dependencies used by the project, along with their respective licenses and compatibility assessment against the MIT License.

### Compatibility Legend

| Symbol | Meaning |
|:------:|:--------|
| ✅ | Fully compatible with MIT — no additional obligations |
| ⚠️ | Compatible with MIT — minor obligations (attribution, file-level copyleft) |
| ❓ | License not automatically detected — manually verified compatible |

### Summary Statistics

| Ecosystem | Total Dependencies | MIT | Apache-2.0 | BSD-* | ISC | MPL-2.0 | Other |
|:----------|:------------------:|:---:|:----------:|:-----:|:---:|:-------:|:-----:|
| Go | — | — | — | — | — | — | — |
| Node.js | — | — | — | — | — | — | — |

**Overall Assessment**: ✅ All dependencies are compatible with the MIT License. No GPL/AGPL/SSPL dependencies detected.

---

## Go Dependencies

| Package | Version | License |
|---------|---------|---------|
| atomicgo.dev/cursor | v0.2.0 | MIT |
| atomicgo.dev/keyboard | v0.2.9 | MIT |
| atomicgo.dev/schedule | v0.1.0 | MIT |
| cloud.google.com/go | v0.65.0 | Apache-2.0 |
| codeberg.org/go-fonts/liberation | v0.5.0 | UNKNOWN |
| codeberg.org/go-latex/latex | v0.1.0 | UNKNOWN |
| codeberg.org/go-pdf/fpdf | v0.10.0 | MIT |
| dario.cat/mergo | v1.0.0 | UNKNOWN |
| git.sr.ht/~jackmordaunt/go-toast/v2 | v2.0.3 | MIT |
| git.sr.ht/~sbinet/gg | v0.6.0 | UNKNOWN |
| github.com/99designs/go-keychain | v0.0.0-20191008050251-8e49817e8af4 | MIT |
| github.com/99designs/keyring | v1.2.2 | MIT |
| github.com/Masterminds/semver | v1.5.0 | UNKNOWN |
| github.com/MetalBlueberry/go-plotly | v0.7.0 | MIT |
| github.com/Microsoft/go-winio | v0.6.1 | MIT |
| github.com/ProtonMail/go-crypto | v1.1.5 | UNKNOWN |
| github.com/acarl005/stripansi | v0.0.0-20180116102854-5a71ef0e047d | MIT |
| github.com/ajstarks/svgo | v0.0.0-20211024235047-1546f124cd8b | UNKNOWN |
| github.com/alecthomas/chroma/v2 | v2.14.0 | UNKNOWN |
| github.com/andybalholm/brotli | v1.2.0 | UNKNOWN |
| github.com/aymanbagabas/go-osc52/v2 | v2.0.1 | MIT |
| github.com/aymanbagabas/go-udiff | v0.3.1 | UNKNOWN |
| github.com/aymerick/douceur | v0.2.0 | MIT |
| github.com/bep/debounce | v1.2.1 | MIT |
| github.com/bitfield/script | v0.24.0 | MIT |
| github.com/campoy/embedmd | v1.0.0 | Apache-2.0 |
| github.com/charmbracelet/colorprofile | v0.4.3 | MIT |
| github.com/charmbracelet/glamour | v0.8.0 | MIT |
| github.com/charmbracelet/lipgloss | v1.1.0 | MIT |
| github.com/charmbracelet/x/ansi | v0.11.6 | MIT |
| github.com/charmbracelet/x/cellbuf | v0.0.15 | MIT |
| github.com/charmbracelet/x/exp/golden | v0.0.0-20241011142426-46044092ad91 | MIT |
| github.com/charmbracelet/x/term | v0.2.2 | MIT |
| github.com/chewxy/math32 | v1.11.1 | BSD-2-Clause |
| github.com/clipperhouse/displaywidth | v0.11.0 | MIT |
| github.com/clipperhouse/uax29/v2 | v2.7.0 | MIT |
| github.com/cloudflare/circl | v1.3.7 | UNKNOWN |
| github.com/containerd/console | v1.0.3 | Apache-2.0 |
| github.com/cyphar/filepath-securejoin | v0.3.6 | UNKNOWN |
| github.com/danieljoos/wincred | v1.2.3 | MIT |
| github.com/daulet/tokenizers | v1.27.0 | MIT |
| github.com/davecgh/go-spew | v1.1.2-0.20180830191138-d8f796af33cc | ISC |
| github.com/dlclark/regexp2 | v1.11.0 | MIT |
| github.com/dmarkham/enumer | v1.6.1 | UNKNOWN |
| github.com/dustin/go-humanize | v1.0.1 | UNKNOWN |
| github.com/dvsekhvalnov/jose2go | v1.8.0 | MIT |
| github.com/edsrzf/mmap-go | v1.2.0 | UNKNOWN |
| github.com/eliben/go-sentencepiece | v0.7.0 | Apache-2.0 |
| github.com/emirpasic/gods | v1.18.1 | UNKNOWN |
| github.com/erkkah/margaid | v0.3.0 | ISC |
| github.com/flytam/filenamify | v1.2.0 | MIT |
| github.com/fsnotify/fsnotify | v1.9.0 | UNKNOWN |
| github.com/go-errors/errors | v1.5.1 | UNKNOWN |
| github.com/go-git/gcfg | v1.5.1-0.20230307220236-3a3c6141e376 | UNKNOWN |
| github.com/go-git/go-billy/v5 | v5.6.2 | Apache-2.0 |
| github.com/go-git/go-git/v5 | v5.13.2 | Apache-2.0 |
| github.com/go-logr/logr | v1.4.3 | Apache-2.0 |
| github.com/go-ole/go-ole | v1.3.0 | MIT |
| github.com/godbus/dbus | v0.0.0-20190726142602-4481cbc300e2 | UNKNOWN |
| github.com/godbus/dbus/v5 | v5.2.2 | UNKNOWN |
| github.com/gofrs/flock | v0.13.0 | UNKNOWN |
| github.com/gofrs/uuid | v4.4.0+incompatible | UNKNOWN |
| github.com/golang/freetype | v0.0.0-20170609003504-e2365dfdc4a0 | UNKNOWN |
| github.com/golang/groupcache | v0.0.0-20210331224755-41bb18bfe9da | Apache-2.0 |
| github.com/golang/protobuf | v1.5.0 | UNKNOWN |
| github.com/gomlx/bsplines | v0.2.0 | Apache-2.0 |
| github.com/gomlx/exceptions | v0.0.3 | Apache-2.0 |
| github.com/gomlx/go-huggingface | v0.3.5 | Apache-2.0 |
| github.com/gomlx/go-xla | v0.2.2 | Apache-2.0 |
| github.com/gomlx/gomlx | v0.27.3 | Apache-2.0 |
| github.com/gomlx/onnx-gomlx | v0.4.2 | Apache-2.0 |
| github.com/google/go-cmp | v0.7.0 | UNKNOWN |
| github.com/google/pprof | v0.0.0-20250317173921-a4b03ec1a45e | Apache-2.0 |
| github.com/google/shlex | v0.0.0-20191202100458-e7afc7fbc510 | Apache-2.0 |
| github.com/google/subcommands | v1.2.0 | Apache-2.0 |
| github.com/google/uuid | v1.6.0 | UNKNOWN |
| github.com/google/wire | v0.7.0 | Apache-2.0 |
| github.com/gookit/color | v1.5.4 | MIT |
| github.com/gorilla/css | v1.0.1 | UNKNOWN |
| github.com/gorilla/websocket | v1.5.3 | UNKNOWN |
| github.com/gsterjov/go-libsecret | v0.0.0-20161001094733-a6f4afe4910c | MIT |
| github.com/hashicorp/golang-lru/v2 | v2.0.7 | MPL-2.0 |
| github.com/itchyny/gojq | v0.12.13 | MIT |
| github.com/itchyny/timefmt-go | v0.1.5 | MIT |
| github.com/jackmordaunt/icns | v1.0.0 | MIT |
| github.com/janpfeifer/go-benchmarks | v0.1.1 | MIT |
| github.com/janpfeifer/gonb | v0.11.3 | MIT |
| github.com/janpfeifer/must | v0.2.0 | Apache-2.0 |
| github.com/jaypipes/ghw | v0.21.3 | UNKNOWN |
| github.com/jaypipes/pcidb | v1.1.1 | Apache-2.0 |
| github.com/jbenet/go-context | v0.0.0-20150711004518-d14ea06fba99 | MIT |
| github.com/jchv/go-winloader | v0.0.0-20250406163304-c1995be93bd1 | ISC |
| github.com/kevinburke/ssh_config | v1.2.0 | MIT |
| github.com/klauspost/compress | v1.18.5 | MIT |
| github.com/knights-analytics/hugot | v0.7.4 | Apache-2.0 |
| github.com/knights-analytics/ortgenai | v0.3.1 | MIT |
| github.com/kr/pretty | v0.3.1 | UNKNOWN |
| github.com/kr/pty | v1.1.1 | UNKNOWN |
| github.com/kr/text | v0.2.0 | UNKNOWN |
| github.com/labstack/echo/v4 | v4.15.2 | MIT |
| github.com/labstack/gommon | v0.5.0 | MIT |
| github.com/leaanthony/clir | v1.3.0 | MIT |
| github.com/leaanthony/debme | v1.2.1 | MIT |
| github.com/leaanthony/go-ansi-parser | v1.6.1 | MIT |
| github.com/leaanthony/gosod | v1.0.4 | MIT |
| github.com/leaanthony/slicer | v1.6.0 | MIT |
| github.com/leaanthony/u | v1.1.1 | MIT |
| github.com/leaanthony/winicon | v1.0.0 | MIT |
| github.com/lithammer/fuzzysearch | v1.1.8 | MIT |
| github.com/lucasb-eyer/go-colorful | v1.3.0 | UNKNOWN |
| github.com/matryer/is | v1.4.1 | MIT |
| github.com/mattn/go-colorable | v0.1.14 | MIT |
| github.com/mattn/go-isatty | v0.0.22 | MIT |
| github.com/mattn/go-runewidth | v0.0.21 | MIT |
| github.com/microcosm-cc/bluemonday | v1.0.27 | UNKNOWN |
| github.com/mitchellh/colorstring | v0.0.0-20190213212951-d06e56a500db | MIT |
| github.com/mtibben/percent | v0.2.1 | MIT |
| github.com/muesli/reflow | v0.3.0 | MIT |
| github.com/muesli/termenv | v0.16.0 | MIT |
| github.com/mutecomm/go-sqlcipher | v0.0.0-20190227152316-55dbde17881f | UNKNOWN |
| github.com/ncruces/go-strftime | v1.0.0 | MIT |
| github.com/nfnt/resize | v0.0.0-20180221191011-83c6a9932646 | UNKNOWN |
| github.com/niemeyer/pretty | v0.0.0-20200227124842-a10e7caefd8e | UNKNOWN |
| github.com/parquet-go/bitpack | v1.0.0 | Apache-2.0 |
| github.com/parquet-go/jsonlite | v1.5.0 | MIT |
| github.com/parquet-go/parquet-go | v0.29.0 | Apache-2.0 |
| github.com/pascaldekloe/name | v1.0.0 | UNKNOWN |
| github.com/pierrec/lz4/v4 | v4.1.26 | UNKNOWN |
| github.com/pjbgf/sha1cd | v0.3.2 | Apache-2.0 |
| github.com/pkg/browser | v0.0.0-20240102092130-5ac0b6a4141c | UNKNOWN |
| github.com/pkg/errors | v0.9.1 | UNKNOWN |
| github.com/pmezard/go-difflib | v1.0.1-0.20181226105442-5d4384ee4fb2 | UNKNOWN |
| github.com/pterm/pterm | v0.12.80 | MIT |
| github.com/remyoudompheng/bigfft | v0.0.0-20230129092748-24d4a6f8daec | UNKNOWN |
| github.com/rivo/uniseg | v0.4.7 | MIT |
| github.com/rogpeppe/go-internal | v1.14.1 | UNKNOWN |
| github.com/sabhiram/go-gitignore | v0.0.0-20210923224102-525f6e181f06 | MIT |
| github.com/samber/lo | v1.53.0 | MIT |
| github.com/schollz/progressbar/v3 | v3.19.0 | MIT |
| github.com/sergi/go-diff | v1.3.2-0.20230802210424-5b0b94c5c0d3 | UNKNOWN |
| github.com/skeema/knownhosts | v1.3.0 | Apache-2.0 |
| github.com/streadway/quantile | v0.0.0-20220407130108-4246515d968d | UNKNOWN |
| github.com/stretchr/objx | v0.5.2 | MIT |
| github.com/stretchr/testify | v1.11.1 | MIT |
| github.com/tc-hib/winres | v0.3.1 | UNKNOWN |
| github.com/tidwall/gjson | v1.14.2 | MIT |
| github.com/tidwall/match | v1.1.1 | MIT |
| github.com/tidwall/pretty | v1.2.0 | MIT |
| github.com/tidwall/sjson | v1.2.5 | MIT |
| github.com/tkrajina/go-reflector | v0.5.8 | Apache-2.0 |
| github.com/twpayne/go-geom | v1.6.1 | UNKNOWN |
| github.com/valyala/bytebufferpool | v1.0.0 | MIT |
| github.com/valyala/fasttemplate | v1.2.2 | MIT |
| github.com/viant/afs | v1.30.0 | Apache-2.0 |
| github.com/viant/sqlite-vec | v0.3.0 | Apache-2.0 |
| github.com/viant/toolbox | v0.34.6-0.20221112031702-3e7cdde7f888 | Apache-2.0 |
| github.com/viant/vec | v0.1.1-0.20240628004145-aad750556278 | Apache-2.0 |
| github.com/viant/xreflect | v0.0.0-20230303201326-f50afb0feb0d | UNKNOWN |
| github.com/viant/xunsafe | v0.9.2 | Apache-2.0 |
| github.com/wailsapp/go-webview2 | v1.0.23 | MIT |
| github.com/wailsapp/mimetype | v1.4.1 | MIT |
| github.com/wailsapp/wails/v2 | v2.12.0 | MIT |
| github.com/wzshiming/ctc | v1.2.3 | MIT |
| github.com/wzshiming/winseq | v0.0.0-20200112104235-db357dc107ae | MIT |
| github.com/x448/float16 | v0.8.4 | MIT |
| github.com/xanzy/ssh-agent | v0.3.3 | Apache-2.0 |
| github.com/xo/terminfo | v0.0.0-20220910002029-abceb7e1c41e | MIT |
| github.com/yalue/onnxruntime_go | v1.30.1 | UNKNOWN |
| github.com/yuin/goldmark | v1.7.4 | MIT |
| github.com/yuin/goldmark-emoji | v1.0.3 | MIT |
| github.com/yusufpapurcu/wmi | v1.2.4 | MIT |
| golang.org/x/crypto | v0.52.0 | UNKNOWN |
| golang.org/x/exp | v0.0.0-20260508232706-74f9aab9d74a | UNKNOWN |
| golang.org/x/image | v0.43.0 | UNKNOWN |
| golang.org/x/mod | v0.36.0 | UNKNOWN |
| golang.org/x/net | v0.55.0 | UNKNOWN |
| golang.org/x/sync | v0.21.0 | UNKNOWN |
| golang.org/x/sys | v0.46.0 | UNKNOWN |
| golang.org/x/term | v0.43.0 | UNKNOWN |
| golang.org/x/text | v0.38.0 | UNKNOWN |
| golang.org/x/time | v0.15.0 | UNKNOWN |
| golang.org/x/tools | v0.45.0 | UNKNOWN |
| golang.org/x/tools/go/expect | v0.1.1-deprecated | UNKNOWN |
| golang.org/x/tools/go/packages/packagestest | v0.1.1-deprecated | UNKNOWN |
| gonum.org/v1/gonum | v0.16.0 | UNKNOWN |
| gonum.org/v1/plot | v0.15.2 | UNKNOWN |
| google.golang.org/protobuf | v1.36.11 | UNKNOWN |
| gopkg.in/check.v1 | v1.0.0-20201130134442-10cb98267c6c | UNKNOWN |
| gopkg.in/warnings.v0 | v0.1.2 | UNKNOWN |
| gopkg.in/yaml.v2 | v2.4.0 | Apache-2.0 |
| gopkg.in/yaml.v3 | v3.0.1 | MIT |
| howett.net/plist | v1.0.2-0.20250314012144-ee69052608d9 | UNKNOWN |
| k8s.io/klog/v2 | v2.140.0 | Apache-2.0 |
| modernc.org/cc/v4 | v4.28.4 | UNKNOWN |
| modernc.org/ccgo/v4 | v4.34.4 | UNKNOWN |
| modernc.org/fileutil | v1.4.0 | UNKNOWN |
| modernc.org/gc/v2 | v2.6.5 | UNKNOWN |
| modernc.org/gc/v3 | v3.1.3 | UNKNOWN |
| modernc.org/goabi0 | v0.2.0 | UNKNOWN |
| modernc.org/libc | v1.73.4 | UNKNOWN |
| modernc.org/mathutil | v1.7.1 | UNKNOWN |
| modernc.org/memory | v1.11.0 | UNKNOWN |
| modernc.org/opt | v0.2.0 | UNKNOWN |
| modernc.org/sortutil | v1.2.1 | UNKNOWN |
| modernc.org/sqlite | v1.53.0 | UNKNOWN |
| modernc.org/strutil | v1.2.1 | UNKNOWN |
| modernc.org/token | v1.1.0 | UNKNOWN |
| mvdan.cc/sh/v3 | v3.7.0 | UNKNOWN |

## Node.js Dependencies

| Package | Version | License | Repository |
|---------|---------|---------|------------|
| @alloc/quick-lru | 5.2.0 | MIT | https://github.com/sindresorhus/quick-lru |
| @hookform/resolvers | 5.4.0 | MIT | https://github.com/react-hook-form/resolvers |
| @jridgewell/gen-mapping | 0.3.13 | MIT | https://github.com/jridgewell/sourcemaps |
| @jridgewell/resolve-uri | 3.1.2 | MIT | https://github.com/jridgewell/resolve-uri |
| @jridgewell/sourcemap-codec | 1.5.5 | MIT | https://github.com/jridgewell/sourcemaps |
| @jridgewell/trace-mapping | 0.3.31 | MIT | https://github.com/jridgewell/sourcemaps |
| @nodelib/fs.scandir | 2.1.5 | MIT | https://github.com/nodelib/nodelib/tree/master/packages/fs/fs.scandir |
| @nodelib/fs.stat | 2.0.5 | MIT | https://github.com/nodelib/nodelib/tree/master/packages/fs/fs.stat |
| @nodelib/fs.walk | 1.2.8 | MIT | https://github.com/nodelib/nodelib/tree/master/packages/fs/fs.walk |
| @standard-schema/utils | 0.3.0 | MIT | https://github.com/standard-schema/standard-schema |
| @types/debug | 4.1.13 | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped |
| @types/estree-jsx | 1.0.5 | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped |
| @types/estree | 1.0.9 | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped |
| @types/hast | 3.0.4 | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped |
| @types/mdast | 4.0.4 | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped |
| @types/ms | 2.1.0 | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped |
| @types/prop-types | 15.7.15 | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped |
| @types/react | 18.3.30 | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped |
| @types/unist | 2.0.11 | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped |
| @types/unist | 3.0.3 | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped |
| @ungap/structured-clone | 1.3.1 | ISC | https://github.com/ungap/structured-clone |
| any-promise | 1.3.0 | MIT | https://github.com/kevinbeaty/any-promise |
| anymatch | 3.1.3 | ISC | https://github.com/micromatch/anymatch |
| arg | 5.0.2 | MIT | https://github.com/vercel/arg |
| bail | 2.0.2 | MIT | https://github.com/wooorm/bail |
| binary-extensions | 2.3.0 | MIT | https://github.com/sindresorhus/binary-extensions |
| braces | 3.0.3 | MIT | https://github.com/micromatch/braces |
| camelcase-css | 2.0.1 | MIT | https://github.com/stevenvachon/camelcase-css |
| ccount | 2.0.1 | MIT | https://github.com/wooorm/ccount |
| character-entities-html4 | 2.1.0 | MIT | https://github.com/wooorm/character-entities-html4 |
| character-entities-legacy | 3.0.0 | MIT | https://github.com/wooorm/character-entities-legacy |
| character-entities | 2.0.2 | MIT | https://github.com/wooorm/character-entities |
| character-reference-invalid | 2.0.1 | MIT | https://github.com/wooorm/character-reference-invalid |
| chokidar | 3.6.0 | MIT | https://github.com/paulmillr/chokidar |
| class-variance-authority | 0.7.1 | Apache-2.0 | https://github.com/joe-bell/cva |
| clsx | 2.1.1 | MIT | https://github.com/lukeed/clsx |
| comma-separated-tokens | 2.0.3 | MIT | https://github.com/wooorm/comma-separated-tokens |
| commander | 4.1.1 | MIT | https://github.com/tj/commander.js |
| cookie | 1.1.1 | MIT | https://github.com/jshttp/cookie |
| cssesc | 3.0.0 | MIT | https://github.com/mathiasbynens/cssesc |
| csstype | 3.2.3 | MIT | https://github.com/frenic/csstype |
| debug | 4.4.3 | MIT | https://github.com/debug-js/debug |
| decode-named-character-reference | 1.3.0 | MIT | https://github.com/wooorm/decode-named-character-reference |
| dequal | 2.0.3 | MIT | https://github.com/lukeed/dequal |
| devlop | 1.1.0 | MIT | https://github.com/wooorm/devlop |
| didyoumean | 1.2.2 | Apache-2.0 | https://github.com/dcporter/didyoumean.js |
| dlv | 1.1.3 | MIT | https://github.com/developit/dlv |
| es-errors | 1.3.0 | MIT | https://github.com/ljharb/es-errors |
| escape-string-regexp | 5.0.0 | MIT | https://github.com/sindresorhus/escape-string-regexp |
| estree-util-is-identifier-name | 3.0.0 | MIT | https://github.com/syntax-tree/estree-util-is-identifier-name |
| extend | 3.0.2 | MIT | https://github.com/justmoon/node-extend |
| fast-glob | 3.3.3 | MIT | https://github.com/mrmlnc/fast-glob |
| fastq | 1.20.1 | ISC | https://github.com/mcollina/fastq |
| fdir | 6.5.0 | MIT | https://github.com/thecodrr/fdir |
| fill-range | 7.1.1 | MIT | https://github.com/jonschlinkert/fill-range |
| function-bind | 1.1.2 | MIT | https://github.com/Raynos/function-bind |
| glob-parent | 5.1.2 | ISC | https://github.com/gulpjs/glob-parent |
| glob-parent | 6.0.2 | ISC | https://github.com/gulpjs/glob-parent |
| hasown | 2.0.4 | MIT | https://github.com/inspect-js/hasOwn |
| hast-util-to-jsx-runtime | 2.3.6 | MIT | https://github.com/syntax-tree/hast-util-to-jsx-runtime |
| hast-util-whitespace | 3.0.0 | MIT | https://github.com/syntax-tree/hast-util-whitespace |
| html-url-attributes | 3.0.1 | MIT | https://github.com/rehypejs/rehype-minify/tree/main/packages/html-url-attributes |
| inline-style-parser | 0.2.7 | MIT | https://github.com/remarkablemark/inline-style-parser |
| is-alphabetical | 2.0.1 | MIT | https://github.com/wooorm/is-alphabetical |
| is-alphanumerical | 2.0.1 | MIT | https://github.com/wooorm/is-alphanumerical |
| is-binary-path | 2.1.0 | MIT | https://github.com/sindresorhus/is-binary-path |
| is-core-module | 2.16.2 | MIT | https://github.com/inspect-js/is-core-module |
| is-decimal | 2.0.1 | MIT | https://github.com/wooorm/is-decimal |
| is-extglob | 2.1.1 | MIT | https://github.com/jonschlinkert/is-extglob |
| is-glob | 4.0.3 | MIT | https://github.com/micromatch/is-glob |
| is-hexadecimal | 2.0.1 | MIT | https://github.com/wooorm/is-hexadecimal |
| is-number | 7.0.0 | MIT | https://github.com/jonschlinkert/is-number |
| is-plain-obj | 4.1.0 | MIT | https://github.com/sindresorhus/is-plain-obj |
| jiti | 1.21.7 | MIT | https://github.com/unjs/jiti |
| js-tokens | 4.0.0 | MIT | https://github.com/lydell/js-tokens |
| lilconfig | 3.1.3 | MIT | https://github.com/antonk52/lilconfig |
| lines-and-columns | 1.2.4 | MIT | https://github.com/eventualbuddha/lines-and-columns |
| longest-streak | 3.1.0 | MIT | https://github.com/wooorm/longest-streak |
| loose-envify | 1.4.0 | MIT | https://github.com/zertosh/loose-envify |
| lucide-react | 1.22.0 | ISC | https://github.com/lucide-icons/lucide |
| markdown-table | 3.0.4 | MIT | https://github.com/wooorm/markdown-table |
| mdast-util-find-and-replace | 3.0.2 | MIT | https://github.com/syntax-tree/mdast-util-find-and-replace |
| mdast-util-from-markdown | 2.0.3 | MIT | https://github.com/syntax-tree/mdast-util-from-markdown |
| mdast-util-gfm-autolink-literal | 2.0.1 | MIT | https://github.com/syntax-tree/mdast-util-gfm-autolink-literal |
| mdast-util-gfm-footnote | 2.1.0 | MIT | https://github.com/syntax-tree/mdast-util-gfm-footnote |
| mdast-util-gfm-strikethrough | 2.0.0 | MIT | https://github.com/syntax-tree/mdast-util-gfm-strikethrough |
| mdast-util-gfm-table | 2.0.0 | MIT | https://github.com/syntax-tree/mdast-util-gfm-table |
| mdast-util-gfm-task-list-item | 2.0.0 | MIT | https://github.com/syntax-tree/mdast-util-gfm-task-list-item |
| mdast-util-gfm | 3.1.0 | MIT | https://github.com/syntax-tree/mdast-util-gfm |
| mdast-util-mdx-expression | 2.0.1 | MIT | https://github.com/syntax-tree/mdast-util-mdx-expression |
| mdast-util-mdx-jsx | 3.2.0 | MIT | https://github.com/syntax-tree/mdast-util-mdx-jsx |
| mdast-util-mdxjs-esm | 2.0.1 | MIT | https://github.com/syntax-tree/mdast-util-mdxjs-esm |
| mdast-util-phrasing | 4.1.0 | MIT | https://github.com/syntax-tree/mdast-util-phrasing |
| mdast-util-to-hast | 13.2.1 | MIT | https://github.com/syntax-tree/mdast-util-to-hast |
| mdast-util-to-markdown | 2.1.2 | MIT | https://github.com/syntax-tree/mdast-util-to-markdown |
| mdast-util-to-string | 4.0.0 | MIT | https://github.com/syntax-tree/mdast-util-to-string |
| medmemo-web | 1.1.10 | UNLICENSED |  |
| merge2 | 1.4.1 | MIT | https://github.com/teambition/merge2 |
| micromark-core-commonmark | 2.0.3 | MIT | https://github.com/micromark/micromark/tree/main/packages/micromark-core-commonmark |
| micromark-extension-gfm-autolink-literal | 2.1.0 | MIT | https://github.com/micromark/micromark-extension-gfm-autolink-literal |
| micromark-extension-gfm-footnote | 2.1.0 | MIT | https://github.com/micromark/micromark-extension-gfm-footnote |
| micromark-extension-gfm-strikethrough | 2.1.0 | MIT | https://github.com/micromark/micromark-extension-gfm-strikethrough |
| micromark-extension-gfm-table | 2.1.1 | MIT | https://github.com/micromark/micromark-extension-gfm-table |
| micromark-extension-gfm-tagfilter | 2.0.0 | MIT | https://github.com/micromark/micromark-extension-gfm-tagfilter |
| micromark-extension-gfm-task-list-item | 2.1.0 | MIT | https://github.com/micromark/micromark-extension-gfm-task-list-item |
| micromark-extension-gfm | 3.0.0 | MIT | https://github.com/micromark/micromark-extension-gfm |
| micromark-factory-destination | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main/packages/micromark-factory-destination |
| micromark-factory-label | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main/packages/micromark-factory-label |
| micromark-factory-space | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main/packages/micromark-factory-space |
| micromark-factory-title | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main/packages/micromark-factory-title |
| micromark-factory-whitespace | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main/packages/micromark-factory-whitespace |
| micromark-util-character | 2.1.1 | MIT | https://github.com/micromark/micromark/tree/main/packages/micromark-util-character |
| micromark-util-chunked | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main/packages/micromark-util-chunked |
| micromark-util-classify-character | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main/packages/micromark-util-classify-character |
| micromark-util-combine-extensions | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main/packages/micromark-util-combine-extensions |
| micromark-util-decode-numeric-character-reference | 2.0.2 | MIT | https://github.com/micromark/micromark/tree/main/packages/micromark-util-decode-numeric-character-reference |
| micromark-util-decode-string | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main/packages/micromark-util-decode-string |
| micromark-util-encode | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main/packages/micromark-util-encode |
| micromark-util-html-tag-name | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main/packages/micromark-util-html-tag-name |
| micromark-util-normalize-identifier | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main/packages/micromark-util-normalize-identifier |
| micromark-util-resolve-all | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main/packages/micromark-util-resolve-all |
| micromark-util-sanitize-uri | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main/packages/micromark-util-sanitize-uri |
| micromark-util-subtokenize | 2.1.0 | MIT | https://github.com/micromark/micromark/tree/main/packages/micromark-util-subtokenize |
| micromark-util-symbol | 2.0.1 | MIT | https://github.com/micromark/micromark/tree/main/packages/micromark-util-symbol |
| micromark-util-types | 2.0.2 | MIT | https://github.com/micromark/micromark/tree/main/packages/micromark-util-types |
| micromark | 4.0.2 | MIT | https://github.com/micromark/micromark/tree/main/packages/micromark |
| micromatch | 4.0.8 | MIT | https://github.com/micromatch/micromatch |
| ms | 2.1.3 | MIT | https://github.com/vercel/ms |
| mz | 2.7.0 | MIT | https://github.com/normalize/mz |
| nanoid | 3.3.12 | MIT | https://github.com/ai/nanoid |
| normalize-path | 3.0.0 | MIT | https://github.com/jonschlinkert/normalize-path |
| object-assign | 4.1.1 | MIT | https://github.com/sindresorhus/object-assign |
| object-hash | 3.0.0 | MIT | https://github.com/puleos/object-hash |
| parse-entities | 4.0.2 | MIT | https://github.com/wooorm/parse-entities |
| path-parse | 1.0.7 | MIT | https://github.com/jbgutierrez/path-parse |
| picocolors | 1.1.1 | ISC | https://github.com/alexeyraspopov/picocolors |
| picomatch | 2.3.2 | MIT | https://github.com/micromatch/picomatch |
| picomatch | 4.0.4 | MIT | https://github.com/micromatch/picomatch |
| pify | 2.3.0 | MIT | https://github.com/sindresorhus/pify |
| pirates | 4.0.7 | MIT | https://github.com/danez/pirates |
| postcss-import | 15.1.0 | MIT | https://github.com/postcss/postcss-import |
| postcss-js | 4.1.0 | MIT | https://github.com/postcss/postcss-js |
| postcss-load-config | 6.0.1 | MIT | https://github.com/postcss/postcss-load-config |
| postcss-nested | 6.2.0 | MIT | https://github.com/postcss/postcss-nested |
| postcss-selector-parser | 6.1.2 | MIT | https://github.com/postcss/postcss-selector-parser |
| postcss-value-parser | 4.2.0 | MIT | https://github.com/TrySound/postcss-value-parser |
| postcss | 8.5.16 | MIT | https://github.com/postcss/postcss |
| prismjs | 1.30.0 | MIT | https://github.com/PrismJS/prism |
| property-information | 7.2.0 | MIT | https://github.com/wooorm/property-information |
| queue-microtask | 1.2.3 | MIT | https://github.com/feross/queue-microtask |
| react-dom | 18.3.1 | MIT | https://github.com/facebook/react |
| react-hook-form | 7.81.0 | MIT | https://github.com/react-hook-form/react-hook-form |
| react-markdown | 10.1.0 | MIT | https://github.com/remarkjs/react-markdown |
| react-router-dom | 7.18.1 | MIT | https://github.com/remix-run/react-router |
| react-router | 7.18.1 | MIT | https://github.com/remix-run/react-router |
| react-virtuoso | 4.18.10 | MIT | https://github.com/petyosi/react-virtuoso |
| react | 18.3.1 | MIT | https://github.com/facebook/react |
| read-cache | 1.0.0 | MIT | https://github.com/TrySound/read-cache |
| readdirp | 3.6.0 | MIT | https://github.com/paulmillr/readdirp |
| remark-gfm | 4.0.1 | MIT | https://github.com/remarkjs/remark-gfm |
| remark-parse | 11.0.0 | MIT | https://github.com/remarkjs/remark/tree/main/packages/remark-parse |
| remark-rehype | 11.1.2 | MIT | https://github.com/remarkjs/remark-rehype |
| remark-stringify | 11.0.0 | MIT | https://github.com/remarkjs/remark/tree/main/packages/remark-stringify |
| resolve | 1.22.12 | MIT | https://github.com/browserify/resolve |
| reusify | 1.1.0 | MIT | https://github.com/mcollina/reusify |
| run-parallel | 1.2.0 | MIT | https://github.com/feross/run-parallel |
| scheduler | 0.23.2 | MIT | https://github.com/facebook/react |
| set-cookie-parser | 2.7.2 | MIT | https://github.com/nfriedly/set-cookie-parser |
| source-map-js | 1.2.1 | BSD-3-Clause | https://github.com/7rulnik/source-map-js |
| space-separated-tokens | 2.0.2 | MIT | https://github.com/wooorm/space-separated-tokens |
| stringify-entities | 4.0.4 | MIT | https://github.com/wooorm/stringify-entities |
| style-to-js | 1.1.21 | MIT | https://github.com/remarkablemark/style-to-js |
| style-to-object | 1.0.14 | MIT | https://github.com/remarkablemark/style-to-object |
| sucrase | 3.35.1 | MIT | https://github.com/alangpierce/sucrase |
| supports-preserve-symlinks-flag | 1.0.0 | MIT | https://github.com/inspect-js/node-supports-preserve-symlinks-flag |
| tailwindcss-animate | 1.0.7 | MIT |  |
| tailwindcss | 3.4.19 | MIT | https://github.com/tailwindlabs/tailwindcss.git#v3 |
| thenify-all | 1.6.0 | MIT | https://github.com/thenables/thenify-all |
| thenify | 3.3.1 | MIT | https://github.com/thenables/thenify |
| tinyglobby | 0.2.17 | MIT | https://github.com/SuperchupuDev/tinyglobby |
| to-regex-range | 5.0.1 | MIT | https://github.com/micromatch/to-regex-range |
| trim-lines | 3.0.1 | MIT | https://github.com/wooorm/trim-lines |
| trough | 2.2.0 | MIT | https://github.com/wooorm/trough |
| ts-interface-checker | 0.1.13 | Apache-2.0 | https://github.com/gristlabs/ts-interface-checker |
| unified | 11.0.5 | MIT | https://github.com/unifiedjs/unified |
| unist-util-is | 6.0.1 | MIT | https://github.com/syntax-tree/unist-util-is |
| unist-util-position | 5.0.0 | MIT | https://github.com/syntax-tree/unist-util-position |
| unist-util-stringify-position | 4.0.0 | MIT | https://github.com/syntax-tree/unist-util-stringify-position |
| unist-util-visit-parents | 6.0.2 | MIT | https://github.com/syntax-tree/unist-util-visit-parents |
| unist-util-visit | 5.1.0 | MIT | https://github.com/syntax-tree/unist-util-visit |
| util-deprecate | 1.0.2 | MIT | https://github.com/TooTallNate/util-deprecate |
| vfile-message | 4.0.3 | MIT | https://github.com/vfile/vfile-message |
| vfile | 6.0.3 | MIT | https://github.com/vfile/vfile |
| zod | 4.4.3 | MIT | https://github.com/colinhacks/zod |
| zustand | 5.0.13 | MIT | https://github.com/pmndrs/zustand |
| zwitch | 2.0.4 | MIT | https://github.com/wooorm/zwitch |

