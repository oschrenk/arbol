#!/usr/bin/env bash
set -euo pipefail

# Generate arbol config from existing ~/Projects directory structure
# Scans up to 3 levels deep for git repositories
# Uses "/" suffix for paths that have both direct repos and subpaths

ROOT="$HOME/Projects"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/arbol"
CONFIG_FILE="$CONFIG_DIR/config.toml"

mkdir -p "$CONFIG_DIR"

# Associative array to group repos by path
declare -A repos_by_path

# Find git repos up to 3 levels deep
while IFS= read -r git_dir; do
    repo_dir="$(dirname "$git_dir")"
    rel_path="${repo_dir#$ROOT/}"

    # Get remote URL
    url="$(git -C "$repo_dir" remote get-url origin 2>/dev/null || echo "")"
    if [[ -z "$url" ]]; then
        echo "Skipping $rel_path (no origin remote)" >&2
        continue
    fi

    # Split path into segments
    IFS='/' read -ra segments <<< "$rel_path"
    num_segments="${#segments[@]}"

    if [[ $num_segments -eq 1 ]]; then
        # Root level repo: repos.name
        tree_path="${segments[0]}"
        repo_name=""
    else
        # Nested repo: repos.path.to = [{ url, name }]
        tree_path="$(IFS='.'; echo "${segments[*]:0:$((num_segments-1))}")"
        repo_name="${segments[$((num_segments-1))]}"
    fi

    # Extract repo name from URL (remove .git suffix and get basename)
    url_repo_name="$(basename "$url" .git)"

    # Build repo entry - only include name if it differs from URL repo name
    if [[ -z "$repo_name" ]] || [[ "$repo_name" == "$url_repo_name" ]]; then
        entry="{ url = \"$url\" }"
    else
        entry="{ url = \"$url\", name = \"$repo_name\" }"
    fi

    # Append to path group
    if [[ -v repos_by_path["$tree_path"] ]]; then
        repos_by_path["$tree_path"]+="|$entry"
    else
        repos_by_path["$tree_path"]="$entry"
    fi

done < <(find "$ROOT" -maxdepth 4 -type d -name ".git" 2>/dev/null | sort)

# Collect all paths
all_paths=()
for path in "${!repos_by_path[@]}"; do
    all_paths+=("$path")
done

# Function to check if a path needs "/" suffix
# (i.e., if it's a prefix of another path)
needs_slash_suffix() {
    local check_path="$1"
    for other_path in "${all_paths[@]}"; do
        if [[ "$other_path" != "$check_path" ]] && [[ "$other_path" == "$check_path."* ]]; then
            return 0  # true - needs suffix
        fi
    done
    return 1  # false - no suffix needed
}

# Write config file
{
    echo "[accounts.default]"
    echo "default = true"
    echo "root = \"~/Projects\""
    echo ""

    # Sort paths and output
    for path in $(printf '%s\n' "${all_paths[@]}" | sort); do
        entries="${repos_by_path[$path]}"

        # Check if this path needs "/" suffix
        if needs_slash_suffix "$path"; then
            echo -n "repos.$path.\"/\" = ["
        else
            echo -n "repos.$path = ["
        fi

        # Convert pipe-separated entries to comma-separated
        first=true
        IFS='|' read -ra entry_list <<< "$entries"
        for entry in "${entry_list[@]}"; do
            if [[ "$first" == true ]]; then
                first=false
            else
                echo -n ", "
            fi
            echo -n "$entry"
        done

        echo "]"
    done
} > "$CONFIG_FILE"

echo "Generated config at $CONFIG_FILE"
