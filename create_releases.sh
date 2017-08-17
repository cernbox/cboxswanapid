#!/usr/bin/env bash
#
# ClawIO build script. Add build variables to compiled binary.
# # Usage:
#
#     $ ./build.bash [output_filename] [git_repo]
#
# Outputs compiled program in current directory.
# Default file name is 'clawiod'.
# Default git repo is current directory.
# Builds always take place from current directory.

set -euo pipefail

: ${output_filename:="${1:-}"}
: ${output_filename:="cboxswanapid"}

: ${git_repo:="${2:-}"}
: ${git_repo:="."}

pkg=main
ldflags=()

# Timestamp of build
ts_name="${pkg}.buildDate"
ts_value=$(date -u +"%a %b %d %H:%M:%S %Z %Y")
ldflags+=("-X" "\"${ts_name}=${ts_value}\"")

# Current tag, if HEAD is on a tag
# This value is used to determine if the current build is a dev build or a release build
# If this value is empty means we are not on an tag, thus is a dev build
current_tag_name="${pkg}.gitTag"
set +e
current_tag_value="$(git -C "${git_repo}" describe --exact-match HEAD 2>/dev/null)"
set -e
ldflags+=("-X" "\"${current_tag_name}=${current_tag_value}\"")

# Nearest tag on branch
tag_name="${pkg}.gitNearestTag"
tag_value="$(git -C "${git_repo}" describe --abbrev=0 --tags HEAD)"
ldflags+=("-X" "\"${tag_name}=${tag_value}\"")

# Commit SHA
commit_name="${pkg}.gitCommit"
commit_value="$(git -C "${git_repo}" rev-parse --short HEAD)"
ldflags+=("-X" "\"${commit_name}=${commit_value}\"")

# Application name
app_name="${pkg}.appName"
ldflags+=("-X" "\"${app_name}=${output_filename}\"")


releases_dir=${git_repo}/releases
rm -rf ${releases_dir}
mkdir -p ${releases_dir}

os=( "linux" "darwin" "windows" )
arch=( "amd64")

if [[ -z "${current_tag_value}" ]]; then
	# dev build
	current_date=$(date +"%Y%m%d%H%M%S")
	for i in "${os[@]}"; do
		for j in "${arch[@]}"; do
			GOOS=$i GOARCH=$j go build -ldflags "${ldflags[*]}" -o ${releases_dir}/"${output_filename}"-${tag_value}-$i-$j-${current_date}-${commit_value}
		done;
	done;
else
	# release build
	for i in "${os[@]}"; do
		for j in "${arch[@]}"; do
			artifact_folder=${releases_dir}/"${output_filename}"-${tag_value}-$i-$j
			mkdir ${artifact_folder}
			GOOS=$i GOARCH=$j go build -ldflags "${ldflags[*]}" -o ${artifact_folder}/"${output_filename}"
			cp LICENSE ${artifact_folder}
			tar -cvzf "${artifact_folder}".tar.gz "${artifact_folder}"
		done;
	done;
fi
