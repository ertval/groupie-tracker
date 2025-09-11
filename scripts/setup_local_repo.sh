#!/usr/bin/env bash
# setup_local_repo.sh
# Small helper for Git Bash (Windows) to initialize or clone a repo and set per-repo user identity.
# Edit variables in the 'Config' section below, then run:
#   bash scripts/setup_local_repo.sh

set -euo pipefail

### Config - edit these before running ###
# Path where repo will be created or where an existing repo lives (relative or absolute)
REPO_PATH="./myrepo"

# If CLONE_URL is non-empty the script will clone into REPO_PATH instead of init
CLONE_URL=""  # e.g. https://gitea.example.com/owner/repo.git

# Per-repo identity (local only)
USER_NAME="Work Name"
USER_EMAIL="you@work.example.com"

# Default branch name to use locally
DEFAULT_BRANCH="main"

# Create simple .gitignore (true/false)
CREATE_GITIGNORE=true

# Initial README content and commit message
README_TITLE="# ${REPO_PATH##*/}"
INIT_COMMIT_MSG="chore: initial commit"

# Remote URL to add (optional). If empty the script will skip adding/updating remote.
REMOTE_URL=""  # e.g. https://gitea.example.com/owner/repo.git

# If true, attempt to push the initial branch to origin (will prompt for auth)
PUSH_ON_INIT=false

##########################################

echo "-> Repo setup helper starting"

# Create or change into the target directory
if [[ -n "$CLONE_URL" ]]; then
  echo "Cloning $CLONE_URL -> $REPO_PATH"
  git clone "$CLONE_URL" "$REPO_PATH"
  cd "$REPO_PATH"
else
  mkdir -p "$REPO_PATH"
  cd "$REPO_PATH"
  if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "Already a git repository: $(pwd)"
  else
    echo "Initializing new git repository in $(pwd)"
    git init
  fi
fi

# Configure per-repo identity
echo "Setting repo identity: $USER_NAME <$USER_EMAIL>"
git config user.name "$USER_NAME"
git config user.email "$USER_EMAIL"

# Ensure default branch name
echo "Setting default branch to '$DEFAULT_BRANCH' (local)"
git symbolic-ref HEAD refs/heads/$DEFAULT_BRANCH 2>/dev/null || git branch -M $DEFAULT_BRANCH

# Add README and .gitignore if repository has no commits
if ! git rev-parse --verify HEAD >/dev/null 2>&1; then
  echo "Repository has no commits. Creating README and optional .gitignore."
  if [[ ! -f README.md ]]; then
    printf "%s\n" "$README_TITLE" > README.md
    git add README.md
  fi

  if [[ "$CREATE_GITIGNORE" = true ]]; then
    if [[ ! -f .gitignore ]]; then
      cat > .gitignore <<'GITIGNORE'
# Binaries for programs and plugins
*.exe
*.dll
*.so
*.dylib

# compiled object files
*.o
*.a

# Go build
bin/
*.test

# Editor directories and files
.vscode/
.idea/
*.swp

# OS files
Thumbs.db
.DS_Store
GITIGNORE
      git add .gitignore
    fi
  fi

  git commit -m "$INIT_COMMIT_MSG" || true
  echo "Initial commit created."
else
  echo "Repository already has commits — skipping initial commit step."
fi

# Add or update remote if provided
if [[ -n "$REMOTE_URL" ]]; then
  if git remote get-url origin >/dev/null 2>&1; then
    echo "Updating origin to $REMOTE_URL"
    git remote set-url origin "$REMOTE_URL"
  else
    echo "Adding origin -> $REMOTE_URL"
    git remote add origin "$REMOTE_URL"
  fi
  git remote -v
else
  echo "No REMOTE_URL provided — skipping remote setup."
fi

# Optionally push
if [[ "$PUSH_ON_INIT" = true && -n "$REMOTE_URL" ]]; then
  echo "Pushing $DEFAULT_BRANCH to origin (you may be prompted for credentials)"
  git push -u origin "$DEFAULT_BRANCH"
else
  echo "Skipping push (PUSH_ON_INIT=$PUSH_ON_INIT). To push manually: git push -u origin $DEFAULT_BRANCH"
fi

echo "-> Done. Repo path: $(pwd)"
git status --short --branch

exit 0
