load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")
load("@rules_cc//cc:defs.bzl", "cc_proto_library")
load("//tools/go_stateify:defs.bzl", "go_library")

package(licenses = ["notice"])

go_library(
    name = "arch",
    srcs = [
        "aligned.go",
        "arch.go",
        "arch_aarch64.go",
        "arch_amd64.go",
        "arch_amd64.s",
        "arch_arm64.go",
        "arch_state_aarch64.go",
        "arch_state_x86.go",
        "arch_x86.go",
        "auxv.go",
        "signal.go",
        "signal_act.go",
        "signal_amd64.go",
        "signal_arm64.go",
        "signal_info.go",
        "signal_stack.go",
        "stack.go",
        "syscalls_amd64.go",
        "syscalls_arm64.go",
    ],
    importpath = "gvisor.dev/gvisor/pkg/sentry/arch",
    visibility = ["//:sandbox"],
    deps = [
        ":registers_go_proto",
        "//pkg/abi/linux",
        "//pkg/binary",
        "//pkg/cpuid",
        "//pkg/log",
        "//pkg/sentry/context",
        "//pkg/sentry/limits",
        "//pkg/sentry/usermem",
        "//pkg/sync",
        "//pkg/syserror",
    ],
)

proto_library(
    name = "registers_proto",
    srcs = ["registers.proto"],
    visibility = ["//visibility:public"],
)

cc_proto_library(
    name = "registers_cc_proto",
    visibility = ["//visibility:public"],
    deps = [":registers_proto"],
)

go_proto_library(
    name = "registers_go_proto",
    importpath = "gvisor.dev/gvisor/pkg/sentry/arch/registers_go_proto",
    proto = ":registers_proto",
    visibility = ["//visibility:public"],
)
