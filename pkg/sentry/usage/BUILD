load("//tools/go_stateify:defs.bzl", "go_library")

package(licenses = ["notice"])

go_library(
    name = "usage",
    srcs = [
        "cpu.go",
        "io.go",
        "memory.go",
        "memory_unsafe.go",
        "usage.go",
    ],
    importpath = "gvisor.dev/gvisor/pkg/sentry/usage",
    visibility = [
        "//pkg/sentry:internal",
    ],
    deps = [
        "//pkg/bits",
        "//pkg/memutil",
        "//pkg/sync",
    ],
)
