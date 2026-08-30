"""Bazel-managed Linux browser runtime for the pinned Playwright release.

Playwright 1.59 moved x86 Chromium downloads to Chrome for Testing paths that
rules_playwright 0.5.3 does not yet understand. This repository assembles the
exact PLAYWRIGHT_BROWSERS_PATH layout expected by Playwright 1.59.1.
"""

def _playwright_browsers_repository_impl(ctx):
    if ctx.os.name != "linux":
        fail("The Start Playwright browser repository currently supports Linux only")

    arch = ctx.os.arch
    if arch in ["amd64", "x86_64"]:
        chromium_path = "builds/cft/147.0.7727.15/linux64/chrome-linux64.zip"
        headless_path = "builds/cft/147.0.7727.15/linux64/chrome-headless-shell-linux64.zip"
        ffmpeg_path = "builds/ffmpeg/1011/ffmpeg-linux.zip"
        chromium_sha256 = "04883e331b31448d6255fa058f0e1c7e8657c004da1f8a1c04eb28855a37ec6c"
        headless_sha256 = "cf779cd5486f3fc5a18bea4ac606026a1667ae16efaee15af51bd3ad3ae01092"
        ffmpeg_sha256 = "ebc74fc5b94830176a3c2914ae96bd8bc7f6a91f4f33890230f84a172ee61ccc"
    elif arch in ["aarch64", "arm64"]:
        chromium_path = "builds/chromium/1217/chromium-linux-arm64.zip"
        headless_path = "builds/chromium/1217/chromium-headless-shell-linux-arm64.zip"
        ffmpeg_path = "builds/ffmpeg/1011/ffmpeg-linux-arm64.zip"
        chromium_sha256 = ""
        headless_sha256 = ""
        ffmpeg_sha256 = ""
    else:
        fail("Unsupported Linux architecture for Playwright: {}".format(arch))

    base_urls = [
        "https://cdn.playwright.dev",
        "https://cdn.playwright.dev/dbazure/download/playwright",
        "https://playwright.download.prss.microsoft.com/dbazure/download/playwright",
    ]
    ctx.download_and_extract(
        url = [base + "/" + chromium_path for base in base_urls],
        output = "chromium-1217",
        sha256 = chromium_sha256,
    )
    ctx.download_and_extract(
        url = [base + "/" + headless_path for base in base_urls],
        output = "chromium_headless_shell-1217",
        sha256 = headless_sha256,
    )
    ctx.download_and_extract(
        url = [base + "/" + ffmpeg_path for base in base_urls],
        output = "ffmpeg-1011",
        sha256 = ffmpeg_sha256,
    )

    # Playwright uses these markers to distinguish complete installations from
    # interrupted downloads.
    ctx.file("chromium-1217/INSTALLATION_COMPLETE", "")
    ctx.file("chromium_headless_shell-1217/INSTALLATION_COMPLETE", "")
    ctx.file("ffmpeg-1011/INSTALLATION_COMPLETE", "")
    ctx.file("root.marker", "Playwright 1.59.1 browser root\n")
    ctx.file("BUILD.bazel", """
load("@aspect_rules_js//js:defs.bzl", "js_library")
js_library(
    name = "browsers",
    srcs = glob(["**"], exclude = ["BUILD.bazel", "root.marker"]),
    visibility = ["//visibility:public"],
)
js_library(
    name = "browser_root",
    srcs = ["root.marker"],
    visibility = ["//visibility:public"],
)
""")

playwright_browsers_repository = repository_rule(
    implementation = _playwright_browsers_repository_impl,
    attrs = {},
)
