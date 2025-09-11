## Credentials setup and switching — HTTPS 🔒 and SSH 🔑


This guide shows how to create, store and switch between two accounts (example: `work` and `personal`) when pushing to Gitea — and how to create a repo correctly from the start. Commands assume Git Bash (`bash.exe`) on Windows.

What you'll find here
- ✅ Getting started: set commit identity, init or clone a repo, .gitignore, default branch
- ✅ HTTPS: how to save credentials, two clear options for per-repo auth, and how to switch accounts
- ✅ SSH: how to create separate keys, configure host aliases, and use them per-repo
- ✅ Verification, cleanup, and troubleshooting commands

Quick notes
- The username shown in `git remote -v` for HTTPS is the HTTP auth username (not the commit author). Commit author is set with `git config user.name`/`user.email`.
- On Windows prefer the Git Credential Manager (`manager-core`) — it uses OS-secure storage and stores credentials by URL.

---

### Getting started — identity & repo init ⚙️

Before you make commits or push, set your commit identity and create or clone the repository correctly.

1) Set your commit author identity (global or per-repo)

```bash
# Global (applies to all repos unless overridden)
git config --global user.name "Your Name"
git config --global user.email "you@example.com"

# Per-repo (run inside the repository)
git config user.name "Work Name"
git config user.email "you@work.example.com"

# Verify
git config --global --get user.name
git config --global --get user.email
```

2) (Optional) Enable commit signing (GPG or SSH)

```bash
# Example (GPG):
# generate a GPG key, then:
git config --global user.signingkey <KEY_ID>
git config --global commit.gpgsign true

# To disable per-repo
git config commit.gpgsign false
```

3) Create a new repo or clone an existing one

```bash
# Create locally and push to remote later:
mkdir myrepo && cd myrepo
git init
echo "# MyRepo" > README.md
git add README.md
git commit -m "chore: initial commit"

# Or clone an existing remote
git clone https://gitea.example.com/owner/repo.git
```

4) Create a useful `.gitignore` and set your default branch

```bash
# Add a sensible .gitignore for your project then:
git add .gitignore && git commit -m "chore: add .gitignore"

# Set or rename default branch to 'main' locally
git branch -M main
```

5) Add remote and push (first push will authenticate and save credentials)

```bash
git remote add origin https://gitea.example.com/owner/repo.git
git push -u origin main
```

Notes
- Commit author identity (name/email) only affects commit metadata, not network authentication.
- If you use multiple identities (work vs personal) use per-repo config or `includeIf` to load per-folder settings.

---

---

### A. HTTPS (recommended when using tokens) 🛡️

1) Configure a credential helper (one-time)

```bash
# Recommended (Windows): Git Credential Manager
git config --global credential.helper manager-core

# Alternative (short-lived cache):
git config --global credential.helper 'cache --timeout=3600'
```

2) Pick an option for per-repo username (choose one)

2a) Option A — Embed username in the remote URL (explicit, visible)

```bash
# Example (run in the repo):
git remote set-url origin https://workuser@gitea.example.com/workowner/workrepo.git

# Verify the username is visible
git remote -v
```

Pros: easy to see which account is used; credential helpers store credentials per-URL.
Cons: username appears in the URL and in some UIs.

2b) Option B — Keep URL clean, set `credential.username` per-repo

```bash
# Run inside the work repo
git config credential.username workuser

# For a personal repo
git config credential.username personal

# Verify
git config --get credential.username
```

Pros: URL stays clean. Credential helper will use the configured username when matching stored credentials.
Cons: slightly less explicit than the embedded-URL approach.

3) First push (save credentials)

```bash
# Push; you will be prompted for username and password/token and the helper will save them
git push
```

After a successful push the credential helper (e.g. manager-core) will cache/store your credentials tied to the URL or repo's username.

4) Switch which account is used for a repo

- If you used Option A (embedded username): change the remote URL to the other username (see 2a).
- If you used Option B (`credential.username`): update or unset that config and push to re-authenticate.
- Or delete the saved credential for the host so the helper prompts again with the other username.

5) Remove stored HTTPS credentials / cleanup ⚠️

- Use the Windows Credential Manager UI to remove entries for your Gitea host, or use the Git Credential Manager CLI to erase entries.
- To stop using `credential.username` in a repo:

```bash
git config --unset credential.username
```

6) Verify which helper(s) are configured

```bash
git config --show-origin --get-all credential.helper
git config --get-all credential.helper

# In Git Bash, check if manager-core exists
which git-credential-manager-core || where git-credential-manager-core
```

---

### B. SSH (recommended for multiple accounts on the same host) 🧭

1) Generate separate SSH keys (one per account)

```bash
# Work key
ssh-keygen -t ed25519 -f "$HOME/.ssh/id_ed25519_work" -C "work@example.com"

# Personal key
ssh-keygen -t ed25519 -f "$HOME/.ssh/id_ed25519_personal" -C "personal@example.com"
```

2) Add each public key to the matching Gitea account (Gitea UI → Settings → SSH Keys).

3) Create an SSH config with host aliases

Edit (or create) `$HOME/.ssh/config` and add the aliases below:

```text
# Work account alias
Host gitea-work
  HostName gitea.example.com
  User git
  IdentityFile ~/.ssh/id_ed25519_work
  IdentitiesOnly yes

# Personal account alias
Host gitea-personal
  HostName gitea.example.com
  User git
  IdentityFile ~/.ssh/id_ed25519_personal
  IdentitiesOnly yes
```

Notes: `Host` is an alias you use in remote URLs; `HostName` is the real server. `IdentitiesOnly yes` helps ensure the configured key is used.

4) Use the aliases in remotes

```bash
# Work repo
git remote set-url origin git@gitea-work:workowner/workrepo.git

# Personal repo
git remote set-url origin git@gitea-personal:personalowner/personalrepo.git

# Verify
git remote -v
```

5) (Optional) Use `ssh-agent` to cache keys for your session

```bash
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_ed25519_work
ssh-add ~/.ssh/id_ed25519_personal
ssh-add -l
```

6) Verify the SSH connections

```bash
ssh -T git@gitea-work
ssh -T git@gitea-personal
```

You should see a greeting from Gitea for each account.

7) Switch accounts (SSH)

- Change the repo's remote to use the other alias (see step 4). Each alias maps to a different key/account.

---

### C. Quick verification & troubleshooting 🛠️

- Show remotes and URLs:

```bash
git remote -v
git remote show origin
```

- See repo-level credential username (if set):

```bash
git config --get credential.username
```

- See configured credential helper(s):

```bash
git config --show-origin --get-all credential.helper
```

- Remove repo-specific credential setting:

```bash
git config --unset credential.username
```

- Revert a remote URL to no-username form:

```bash
git remote set-url origin https://gitea.example.com/owner/repo.git
```

---

### D. Commit author vs push authentication 📝

Set commit author (this controls who appears as the author in commits — it does NOT control authentication):

```bash
git config --global user.name "Your Name"
git config --global user.email "you@example.com"

# Or per-repo
git config user.name "Work Name"
git config user.email "you@work.example.com"
```

---

### E. Short examples — quick copy/paste ✅

HTTPS (embed username):

```bash
git remote set-url origin https://workuser@gitea.example.com/workowner/workrepo.git
```

HTTPS (credential.username):

```bash
git config credential.username workuser
```

SSH (use host alias):

```bash
git remote set-url origin git@gitea-work:workowner/workrepo.git
```

---

If you'd like, tell me which method you prefer (HTTPS tokens or SSH keys) and I will generate the exact commands for your repositories and host.

Small checklist mapping to your request
- [x] Make HTTPS options clearer (2a / 2b)
- [x] Improve SSH section and host-alias explanation
- [x] Add emojis and tidy troubleshooting

Completion: updated `doc/CREDENTIALS_SETUP.md` for clarity and readability.

---

### F. Ubuntu / Linux notes 🐧

If you run the same process on Ubuntu (or another Linux distro) the steps are the same in principle but there are a few common differences and helper choices to be aware of.

HTTPS credential helpers on Linux
- Git Credential Manager Core (GCM Core) is cross-platform and can be used on Linux if installed. If available use:

```bash
git config --global credential.helper manager-core
```

- Alternatively, many Linux installs use the libsecret helper which stores credentials in the GNOME keyring. To use it you may need to install the helper and then configure Git:

```bash
# Install libsecret helper (Debian/Ubuntu example)
# sudo apt install libsecret-1-0 libsecret-1-dev pkg-config
# then build or install the helper (depends on your distribution)

# Configure Git to use the libsecret helper
git config --global credential.helper /usr/share/doc/git/contrib/credential/libsecret/git-credential-libsecret
```

- If you installed GCM Core via your package manager or the official installer, prefer `manager-core` because it integrates well with modern credential storage.

SSH on Linux
- SSH key generation and `$HOME/.ssh/config` are identical to the Git Bash instructions — on Ubuntu `$HOME` is typically `/home/youruser`.
- Use the system `ssh-agent` (or `gnome-keyring`/`keychain`) to cache keys. Example with `ssh-agent`:

```bash
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_ed25519_work
ssh-add ~/.ssh/id_ed25519_personal
```

Verification and troubleshooting tips on Ubuntu
- Check which credential helper Git will use:

```bash
git config --show-origin --get-all credential.helper
```

- If `manager-core` is not installed and you want it, follow the official Microsoft/GitHub instructions for installing GCM Core on Linux (package or binary). If you prefer not to install extra packages, the `libsecret` helper is the typical choice on GNOME-based systems.

Security note
- On Linux avoid `credential.helper store` unless you understand the security implications — it writes plaintext credentials to `~/.git-credentials`.

---
