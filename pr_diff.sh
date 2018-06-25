#!/bin/bash

set -euo pipefail

if [[ $# -ne 2 ]]; then
    echo "Usage: ./pr_diff <commit-id-1> <commit-id-2>"
    exit 1
fi

readonly github_url="https://github.com/KyberNetwork/reserve-data/pull/"

pr_commit_ids=($(git log --oneline "$1".."$2" | awk '/Merge pull request/ {print $1}'))

for commit_id in "${pr_commit_ids[@]}"; do
    git show --format="%B" "$commit_id" |\
	awk -v github_url="$github_url" 'NR == 1 {$4="["$4"]("github_url substr($4,2)")"; print "* "$0; next}; {print "  "$0}'
done
