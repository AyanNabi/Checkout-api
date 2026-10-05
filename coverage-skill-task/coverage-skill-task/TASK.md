# Intern Task — Write a coverage Skill

**Write a Skill: one markdown file that tells an AI how to raise test coverage on a codebase without producing worthless tests. Then prove it works by running it yourself, on the `wallet` module, in Gemini chat.**

That's the task. The Skill is the deliverable; the tests are the evidence that it works.

Tool: Gemini chat. Pick your language: **Python, JavaScript, or Go** — the same module is in all three. 90–120 minutes.

---

## Part 1 — The Skill (the deliverable)

A Skill is a reusable instruction file. Written once, it makes every future run better. Yours must be good enough that **another intern could hand it to a fresh Gemini chat, point it at a different codebase, and get useful tests out.**

Write it as `coverage-skill.md`. It must answer all six of these:

1. **Discover** — how does the agent find what is untested? What commands does it run, what does it read?
2. **Prioritize** — given ten uncovered functions, which does it test first? Say why, in a rule that can be applied without you.
3. **Write** — what does a test worth keeping look like? Be specific enough to be checkable.
4. **Validate** — after each candidate test, what gets run, and what counts as success?
5. **Reject** — the list of things that get thrown away even if coverage went up. This section is the one that separates good submissions from bad ones.
6. **Stop** — when is the loop done? "When coverage is 100%" is not an acceptable answer; say why in your write-up.

**The hard part.** Your Skill must say what to do when a test fails. A failing test means one of two things: the test is wrong, or the code is wrong. An agent left to itself will almost always assume the test is wrong and rewrite it until it passes. Write the rule that stops that.

Heads up: **the `wallet` module may contain real bugs.** Neither the existing tests nor the coverage number will tell you.

---

## Part 2 — Run it (the evidence)

Pick a language folder in `starter/`. Read its `RUN.md` for the commands and your starting coverage.

Then run your own Skill by hand in Gemini chat:

- Paste your Skill at the top of a fresh chat, then the module source.
- Follow your own process. If the Skill says "prioritize by X", actually prioritize by X.
- Run the tests and the coverage command yourself after every batch. Not at the end.
- When your Skill's reject rules say to throw something away, throw it away — and log it.

If you find yourself ignoring your own Skill, that is a finding. Fix the Skill, note what was wrong with it, and say so in the write-up. That correction is worth more than a high coverage number.

---

## Starting coverage

| Language | Baseline | Command |
|---|---|---|
| Python | 46% of statements | `python3 -m coverage run -m pytest -q && python3 -m coverage report -m --include=wallet.py` |
| JavaScript | 61% line / 44% funcs | `node --test --experimental-test-coverage` |
| Go | 38.5% of statements | `go test -cover ./...` |

The three numbers are not comparable — different tools count different things. You are measured on your own delta, not against each other.

A reasonable target is **85%+**, but read the grading section before you chase it.

---

## Submit

1. **`coverage-skill.md`** — the Skill.
2. **Your test files.**
3. **An iteration log** — one line per candidate: what you tried, coverage before → after, kept or discarded, why. Six lines is fine. Zero discards is a red flag.
4. **Final coverage output**, pasted verbatim.
5. **The chat transcript**, complete.
6. **Half a page:**
   - Which of your six sections did you have to rewrite mid-run, and what went wrong?
   - Did any test fail? What did you conclude, and how did you decide?
   - Which kept test do you think is the most valuable, and which is closest to padding?

---

## Grading

**Objective checks we run on your submission:**

- Coverage delta from your language's baseline.
- **Your tests are run against a corrected copy of the module.** If a test of yours fails there, it means you wrote down the module's current buggy behaviour as if it were correct. That costs more than the coverage it earned.
- Whether you reported a bug at all.

**Judged:**

- The Skill's reject rules — specific and checkable, or vague good intentions?
- Whether the iteration log shows anything actually being discarded.
- Whether the Skill would work on a codebase you have never seen. If it names `wallet` functions by hand, it is a script, not a Skill.

**The failure mode this task is built to catch:** 95% coverage, every test passing, and a test suite that locks in a bug. That scores below 70% coverage with the bug found and reported.
