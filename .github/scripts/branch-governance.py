"""Inventory origin branches and optionally open a draft for a reviewed branch."""

import argparse
from collections import defaultdict
from datetime import datetime, timezone
from html import escape
import json
import os
import subprocess


PREFIX = "refs/remotes/origin/"
STALE_DAYS = 60
DIVERGENCE = 50


def run(*args, expected=(0,)):
    result = subprocess.run(args, stdout=subprocess.PIPE, encoding="utf-8", check=False)
    if result.returncode not in expected:
        raise subprocess.CalledProcessError(result.returncode, args)
    return result


def api(repo, *args):
    gh = os.environ.get("GH_BIN", "gh")
    return json.loads(run(gh, "api", f"repos/{repo}/pulls", *args).stdout)


def branches():
    fmt = "%(refname)%00%(symref)%00%(objectname)%00%(tree)%00%(committerdate:iso-strict)"
    refs = run("git", "for-each-ref", f"--format={fmt}", PREFIX).stdout
    result = {}
    for row in refs.splitlines():
        ref, alias, commit, tree, date = row.split("\0")
        if not alias:
            # Python <3.11 cannot parse the trailing "Z" used by git.
            parsed = datetime.fromisoformat(date.replace("Z", "+00:00"))
            result[ref[len(PREFIX):]] = (commit, tree, parsed)
    return result


def pull_index(repo):
    by_head, by_base = defaultdict(list), defaultdict(list)
    pages = api(repo, "--method", "GET", "--paginate", "--slurp", "-f", "state=open", "-F", "per_page=100")
    for page in pages:
        for pr in page:
            number = pr["number"]
            head_repo = pr["head"].get("repo")
            if head_repo and head_repo["full_name"].casefold() == repo.casefold():
                by_head[pr["head"]["ref"]].append(number)
            by_base[pr["base"]["ref"]].append(number)
    return by_head, by_base


def divergence(base, head):
    return tuple(map(int, run("git", "rev-list", "--left-right", "--count", f"{base}...{head}").stdout.split()))


def conflicts(base, head):
    return run("git", "merge-tree", "--write-tree", "--name-only", "--no-messages", base, head,
               expected=(0, 1)).returncode == 1


def already_merged(base, head):
    """True when every branch commit has a patch-equivalent commit on the baseline.

    This is the definitive signal for a branch whose work landed through a
    squash merge: a direct merge still conflicts, so merge-tree alone is not
    enough to declare the branch safe to clean up.
    """
    lines = run("git", "cherry", base, head).stdout.splitlines()
    return bool(lines) and all(line.startswith("-") for line in lines)


def report(repo, default, refs, by_head, by_base):
    base, base_tree, _ = refs[default]
    trees = defaultdict(list)
    for name, (_, tree, _) in refs.items():
        trees[tree].append(name)
    now = datetime.now(timezone.utc)
    print(f"## Branch governance · {len(refs)} branches")
    print(f"\nBaseline <code>{escape(default)}</code> at `{base}`. Inactivity ≥ {STALE_DAYS} days, divergence ≥ {DIVERGENCE} commits on either side.\n")
    print("| Branch | Tip | Behind / ahead | Last commit (UTC) | Open PRs | Review |")
    print("|---|---|---:|---|---|---|")
    for name, (head, tree, date) in refs.items():
        behind, ahead = divergence(base, head)
        dependencies = sorted(set(by_head[name] + by_base[name]))
        links = ", ".join(f"[#{n}](https://github.com/{repo}/pull/{n})" for n in dependencies)
        notes = []
        if name == default:
            notes.append("Default branch")
        elif ahead == 0 or tree == base_tree or already_merged(base, head):
            if ahead == 0:
                notes.append("Merged by ancestry")
            elif tree == base_tree:
                notes.append("Same tree as default")
            else:
                notes.append("All commits patch-equivalent on default")
            notes.append("Retain while PRs use it" if dependencies else "Review for cleanup")
        else:
            notes.append("Has unique commits")
            if conflicts(base, head):
                notes.append("Merge conflict")
            elif not by_head[name]:
                notes.append("Review changes and tests before drafting a PR")
        others = [branch for branch in trees[tree] if branch != name]
        if others:
            notes.append("Same tree as " + ", ".join(f"<code>{escape(branch)}</code>" for branch in others))
        if name != default and (now - date).days >= STALE_DAYS:
            notes.append("Inactive")
        if max(behind, ahead) >= DIVERGENCE:
            notes.append("Highly diverged")
        safe_name = escape(name).replace("|", "&#124;")
        safe_notes = " · ".join(notes).replace("|", "&#124;")
        print(f"| <code>{safe_name}</code> | `{head[:12]}` | {behind} / {ahead} | "
              f"{date.astimezone(timezone.utc):%Y-%m-%d} | {links or '—'} | {safe_notes} |")


def open_draft(repo, default, candidate, refs, by_head):
    if candidate == default or candidate not in refs or candidate.startswith("dependabot/"):
        raise SystemExit("Select an existing same-repository development branch.")
    base, base_tree, _ = refs[default]
    head, tree, _ = refs[candidate]
    if by_head[candidate]:
        raise SystemExit("The branch already has an open pull request.")
    if (tree == base_tree or divergence(base, head)[1] == 0
            or already_merged(base, head) or conflicts(base, head)):
        raise SystemExit("The branch must contain unique changes and merge without conflicts.")
    pr = api(repo, "--method", "POST", "-f", f"head={candidate}", "-f", f"base={default}",
             "-F", "draft=true", "-f", f"title=Review {candidate}", "-f",
             f"body=Branch tip at review: {head}. Review the final diff and verify required CI before marking ready.")
    print(f"Opened draft [#{pr['number']}]({pr['html_url']}).")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--draft", metavar="BRANCH")
    args = parser.parse_args()
    repo = os.environ["GITHUB_REPOSITORY"]
    default = os.environ["DEFAULT_BRANCH"]
    refs = branches()
    if default not in refs:
        raise SystemExit(f"Default branch {default!r} missing from origin. Fetch all branches first.")
    by_head, by_base = pull_index(repo)
    if args.draft is None:
        report(repo, default, refs, by_head, by_base)
    else:
        open_draft(repo, default, args.draft, refs, by_head)


if __name__ == "__main__":
    main()
