// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

variable "DOCKER_BAKE_REGISTRY" {}

variable "DOCKER_BAKE_SUFFIX" {}

variable "DEBUG" {
  default = "0"
}

variable "SLURM_VERSION" {
  default = "24.05.7"
}

function "format_name" {
  params = [stage, version, flavor]
  // Remove [:punct:] from string before joining elements
  // NOTE: function library does not seem to support any character classes, have to regex manually
  result = join("_", [regex_replace(stage, "[!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~.]", ""), regex_replace(version, "[!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~.]", ""), regex_replace(flavor, "[!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~.]", "")])
}

function "format_tag" {
  params = [registry, stage, version, flavor, suffix]
  result = format("%s:%s", join("/", compact(["${registry}", "${stage}"])), join("-", compact(["${version}", "${flavor}", "${suffix}"])))
}

group "default" {
  targets = ["_default"]
}

target "_default" {
  name = format_name("${stage}", "${SLURM_VERSION}", "${flavor}")
  context = "${flavor}/"
  tags = [
    format_tag("${DOCKER_BAKE_REGISTRY}", "${stage}",
      "${length(regexall("^(?<major>[0-9]+)\\.(?<minor>[0-9]+)\\.(?<patch>[0-9]+)(?:-(?<rev>.+))?$", "${SLURM_VERSION}")) > 0 ?
        "${format("%s.%s", "${regex("^(?<major>[0-9]+)\\.(?<minor>[0-9]+)\\.(?<patch>[0-9]+)(?:-(?<rev>.+))?$", "${SLURM_VERSION}")["major"]}", "${regex("^(?<major>[0-9]+)\\.(?<minor>[0-9]+)\\.(?<patch>[0-9]+)(?:-(?<rev>.+))?$", "${SLURM_VERSION}")["minor"]}")}" :
        "${SLURM_VERSION}"}",
      "${flavor}", "${DOCKER_BAKE_SUFFIX}"),
    format_tag("${DOCKER_BAKE_REGISTRY}", "${stage}", "${SLURM_VERSION}", "${flavor}", "${DOCKER_BAKE_SUFFIX}"),
  ]
  matrix = {
    stage = [
      "slurmctld",
      "slurmd",
      "slurmdbd",
      "slurmrestd",
      "sackd",
    ]
    flavor = [
      "rockylinux9",
      "ubuntu24.04",
    ]
  }
  args = {
    SLURM_VERSION = "${SLURM_VERSION}"
    DEBUG = "${DEBUG}"
  }
  target = stage
}

target "rockylinux9" {
  name = format_name("${stage}", "${SLURM_VERSION}", "${flavor}")
  extends = [
    "_default",
  ]
  matrix = {
    stage = [
      "slurmctld",
      "slurmd",
      "slurmdbd",
      "slurmrestd",
      "sackd",
    ]
    flavor = [
      "rockylinux9",
    ]
  }
}

target "ubuntu2404" {
  name = format_name("${stage}", "${SLURM_VERSION}", "${flavor}")
  extends = [
    "_default",
  ]
  matrix = {
    stage = [
      "slurmctld",
      "slurmd",
      "slurmdbd",
      "slurmrestd",
      "sackd",
    ]
    flavor = [
      "ubuntu24.04",
    ]
  }
}
