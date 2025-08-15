---
title: Image Preview
description: Learn how image preview works in superfile and how terminal compatibility is determined.
head:
  - tag: title
    content: Image Preview | superfile
---

This tutorial will teach you how to use superfile’s image preview feature step by step.

## What is Image Preview?

superfile supports image previews directly in your terminal using several display protocols. When supported, images can be shown inline without any external viewer.

---

## Terminal Compatibility

superfile automatically detects your terminal using the `$TERM` and `$TERM_PROGRAM` environment variables. We support rendering on the following terminals:

| Terminal              | Protocol         | Image Preview Support |
|-----------------------|------------------|------------------------|
| **kitty**             | Kitty protocol   | ✅                     |
| **WezTerm**           | Kitty protocol   | ✅                     |
| **Ghostty**           | Kitty protocol   | ✅                     |
| **iTerm2**            | iTerm2 inline    | ✅                     |
| **VSCode**            | iTerm2 inline    | ✅                     |
| **Tabby**             | iTerm2 inline    | ✅                     |
| **Hyper**             | iTerm2 inline    | ✅                     |
| **Mintty**            | iTerm2 inline    | ✅                     |
| **Warp**              | iTerm2 inline    | ✅                     |
| **Rio**               | iTerm2 inline    | ✅                     |
| **Konsole**           | iTerm2 inline    | ✅                     |
| **foot**              | Sixel graphics   | ✅                     |
| **Windows Terminal**  | Sixel graphics   | ✅                     |
| **Black Box**         | Sixel graphics   | ✅                     |
| **xterm**             | Sixel graphics   | ✅                     |

> ✅ means full support for inline image preview using one of the supported protocols
> Protocols are tried in order: Kitty → iTerm2 inline → Sixel → ANSI fallback

---

## Supported Protocols

superfile supports the following rendering protocols and will automatically choose the best one based on your terminal:

| Protocol Name     | Description                                                                                   | Status      |
|-------------------|-----------------------------------------------------------------------------------------------|-------------|
| **Kitty protocol** | Most capable, pixel-accurate rendering with transparency and scaling support.                | ✅ Preferred|
| **Sixel**          | DEC standard graphics protocol with wide terminal support including foot, xterm, Windows Terminal. | ✅ Tertiary |
| **iTerm2 inline**  | iTerm2’s proprietary image format, used in iTerm2, VSCode, Tabby, Hyper, Konsole, etc.     | ✅ Secondary|
| **ANSI**           | Fallback text rendering using ANSI blocks or metadata only.                                  | ✅ Always   |

---

## Terminal Detection and Pixel Size

superfile detects terminal capabilities by inspecting:

- `$TERM`
- `$TERM_PROGRAM`
- Specific environment variables like `$KITTY_WINDOW_ID`, `$ITERM_SESSION_ID`, `$VSCODE_INJECTION`, `$WT_SESSION`
- Terminal feature queries (when supported)

These variables help us decide whether advanced rendering might be possible. The detection system follows Yazi's comprehensive approach with fallback chains to ensure maximum terminal compatibility.

To scale images correctly, superfile sends the following escape code:

```
\x1b[16t
```

This sequence queries the terminal for the size of each **cell in pixels**. superfile uses the result to:

- Maintain correct image aspect ratio
- Avoid distortions in previews
- Adapt to terminal resizes

If your terminal does not support `\x1b[16t`, we fallback to default assumptions like `10×20 px per cell`.

## Graceful Fallback to ANSI

When advanced image preview isn't supported (for example, when the terminal doesn't support any of the graphics protocols), superfile gracefully falls back to an ANSI-based preview using color-coded blocks.

The fallback chain is: **Kitty Protocol** → **iTerm2 Inline Images** → **Sixel Graphics** → **ANSI Blocks**

This ensures a consistent and reliable experience across all terminal environments, from modern terminals with advanced graphics support to legacy terminals with only basic ANSI color support.