load("//tools/go_stateify:defs.bzl", "go_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")
load("@io_bazel_rules_go//go:def.bzl", "go_test")
load("@rules_cc//cc:defs.bzl", "cc_proto_library")

package(licenses = ["notice"])

go_library(
    name = "metric",
    srcs = ["metric.go"],
    importpath = "gvisor.dev/gvisor/pkg/metric",
    visibility = ["//:sandbox"],
    deps = [
        ":metric_go_proto",
        "//pkg/eventchannel",
        "//pkg/log",
        "//pkg/sync",
    ],
)

proto_library(
    name = "metric_proto",
    srcs = ["metric.proto"],
    visibility = ["//:sandbox"],
)

cc_proto_library(
    name = "metric_cc_proto",
    visibility = ["//:sandbox"],
    deps = [":metric_proto"],
)

go_proto_library(
    name = "metric_go_proto",
    importpath = "gvisor.dev/gvisor/pkg/metric/metric_go_proto",
    proto = ":metric_proto",
    visibility = ["//:sandbox"],
)

go_test(
    name = "metric_test",
    srcs = ["metric_test.go"],
    embed = [":metric"],
    deps = [
        ":metric_go_proto",
        "//pkg/eventchannel",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)
