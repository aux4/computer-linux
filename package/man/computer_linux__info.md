#### Description

Describes the provider for `aux4/computer`: `provider` (`linux`), `contract` (the provider contract version, `1`),
`platform` (`linux/amd64`, `linux/arm64` or `linux/386`), the package `version`, the `session` type (`x11`,
`xwayland`, `wayland` or `none`), `capabilities` (booleans: `background`, `accessibility`, `screenshot`, `annotate`,
`ocr`, `ocrScreen`, `input`, `windows`, `clipboard`, `menus`, `dialogs`), the same `permissions` as `permissions`,
and whether the stop switch is on. On Wayland it also lists what is `unsupported`. Read-only; needs no permission.

#### Usage

```bash
aux4 computer linux info
```

#### Example

```bash
aux4 computer linux info
```

```json
{"capabilities":{"accessibility":true,"annotate":true,"background":true,"clipboard":true,"dialogs":true,"input":true,"menus":true,"ocr":true,"ocrScreen":true,"screenshot":true,"windows":true},"contract":1,"ocrEngine":"tesseract","permissions":{"accessibility":{"granted":true,"guidance":"Accessibility (AT-SPI) is available.","toolkitAccessibility":true},"input":{"granted":true,"guidance":"Input through XTEST is available."},"screenRecording":{"granted":true,"guidance":"Screen capture is available (X11)."}},"platform":"linux/amd64","provider":"linux","session":"x11","stopped":false,"version":"0.1.0"}
```
