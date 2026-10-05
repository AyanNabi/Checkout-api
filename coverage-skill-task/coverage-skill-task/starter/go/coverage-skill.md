# Coverage Skill

## Purpose

Increase meaningful automated test coverage in an existing codebase without writing tests whose only purpose is to increase the coverage percentage.

The goal is not maximum coverage. The goal is a test suite that documents and protects important, observable behavior while detecting incorrect implementations.

---

## 1. Discover

Before writing tests, inspect the project and understand how it is tested.

### Step 1: Read the project instructions

Read:

* README files
* test instructions
* build/run instructions
* existing test files
* package/module configuration
* CI configuration if present

Do not invent commands. Use the commands documented by the project.

### Step 2: Establish a baseline

Run the project's existing test suite and coverage command before changing anything.

Record:

* whether tests pass
* coverage percentage
* line/statement coverage
* branch coverage if available
* function coverage if available

Save the baseline output.

### Step 3: Read the implementation

Read the source code covered by the assignment, not only the uncovered line report.

Identify:

* public functions and methods
* error paths
* boundary conditions
* branches
* state transitions
* parsing/validation logic
* calculations
* externally observable side effects
* interactions between functions

### Step 4: Read existing tests

Determine:

* what behavior is already protected
* what important cases are missing
* whether existing tests appear to encode questionable behavior
* whether tests are testing implementation details instead of behavior

### Step 5: Build an uncovered-behavior inventory

For each uncovered or weakly covered area, record:

* function/method
* missing behavior
* relevant branch or boundary
* why the behavior matters
* possible test
* risk that the existing implementation is wrong

Coverage reports are evidence about what was executed. They are not evidence that the behavior is correct.

---

## 2. Prioritize

Do not simply test uncovered lines from top to bottom.

For every candidate, assign priority using this rule:

### Priority = behavioral risk × importance × decision complexity

Test first when the candidate:

1. affects externally visible results;
2. has multiple branches or error paths;
3. contains boundary conditions;
4. performs calculations or transformations;
5. validates/parses user input;
6. controls security, authorization, money, persistence, or state;
7. is used by other important functions.

Prefer a small number of high-value behavioral tests over many tests that merely execute lines.

A candidate that only contains trivial plumbing should be lower priority unless it is required to exercise important behavior.

When two candidates have similar risk, prefer the one with more uncovered branches.

---

## 3. Write

A test is worth keeping only if it has a clear behavioral purpose.

Every candidate test should answer:

> What incorrect behavior would this test catch?

A useful test should have:

* clear setup;
* one primary behavior under test;
* a meaningful input;
* an observable expected result;
* a failure message that helps identify the broken behavior.

Prefer testing the public behavior of a function/module rather than private implementation details.

### Boundary testing

For validation and calculations, test values around boundaries:

* minimum valid value;
* maximum valid value;
* just below the boundary;
* just above the boundary;
* zero;
* empty input where relevant;
* malformed input where relevant.

### Error testing

For every documented error path, test:

* the input that causes the error;
* that an error is actually returned/thrown;
* the relevant error category/message when that is part of the contract.

Do not weaken assertions merely to make a failing test pass.

### Integration behavior

When a public function combines several lower-level operations, test important combinations through the public function.

Do not create dozens of tests that separately duplicate lower-level behavior if one integration test already protects the observable result.

### Mathematical behavior

For calculations, include values that distinguish competing implementations, especially:

* exact boundaries;
* rounding boundaries;
* zero;
* one;
* values just below/above a rounding threshold.

---

## 4. Validate

After each candidate or small logical batch:

1. Run the relevant test.
2. Run the complete test suite.
3. Run the coverage command.
4. Compare coverage with the previous checkpoint.

A candidate is successful only if:

* the test passes;
* the complete suite still passes;
* the test has a clear behavioral purpose;
* the test increases meaningful coverage or strengthens an important existing path;
* the assertion is specific enough to detect an incorrect implementation.

Do not judge success solely by the coverage percentage.

Record each iteration:

```text
candidate | coverage before -> after | kept/discarded | reason
```

Run coverage after every batch, not only at the end.

---

## 5. What to do when a test fails

A failing test must never automatically be rewritten until it passes.

When a new test fails, stop and investigate.

Use this decision process:

### A. Check the test first

Verify:

* the expected result follows from the documented contract;
* the test setup is valid;
* the assertion is correct;
* the test is not accidentally relying on implementation details;
* the test input is actually intended to exercise the behavior.

### B. Check the implementation independently

Read the implementation and compare it with:

* documentation;
* comments;
* existing API behavior;
* mathematical/business rules;
* related tests;
* requirements.

If the implementation contradicts the documented or logically required behavior, classify the result as a possible implementation bug.

### C. Never change the expected value just to make the test pass

If the test describes correct required behavior and the implementation fails it:

* keep the test;
* report the implementation bug;
* do not "fix" the test to match the current implementation.

If the test was genuinely wrong:

* discard or correct the test;
* record why it was wrong.

### Bug rule

**A failing test is evidence to investigate, not evidence that the expected behavior is wrong.**

Never encode currently observed buggy behavior as a passing test merely to increase coverage.

If the assignment does not authorize changing production code, report the bug separately rather than silently modifying the implementation.

---

## 6. Reject

Discard a candidate test even if it increases coverage when it meets any of these conditions:

### 1. It tests implementation details

Examples:

* exact internal variable values;
* private helper calls when public behavior is sufficient;
* loop structure;
* number of iterations;
* internal data structures that are not part of the contract.

### 2. It has no meaningful assertion

Reject tests such as:

```text
call function
assert no panic
```

when the function has an observable result that should be checked.

### 3. It duplicates another test

If two tests exercise the same behavior and one adds no meaningful input, boundary, branch, or failure mode, keep the stronger test and discard the duplicate.

### 4. It only exists for coverage

Reject a test whose only justification is:

> "This line was uncovered."

There must be a behavioral reason for the test.

### 5. It asserts accidental current behavior

If the behavior appears to be a bug or accidental implementation detail, do not lock it into the test suite.

### 6. It has weak assertions

Examples:

* only checking that execution completed;
* checking only that a result is non-null when its value matters;
* checking only one field when the operation's correctness depends on multiple fields.

### 7. It makes the test suite less trustworthy

Reject tests that:

* depend on execution order unnecessarily;
* use uncontrolled external state;
* are flaky;
* contain excessive mocking that hides the behavior being tested;
* make broad assertions unrelated to the purpose of the test.

### 8. It passes only because the test was weakened

Never replace a precise expected result with a vague assertion just to make the test pass.

Every discarded candidate must be recorded in the iteration log with a reason.

---
**## 7. Stop**

Do not stop merely because coverage reaches 100%.

Stop when all of the following are true:

1. High-risk public behavior is covered.

2. Important branches and boundary conditions have meaningful tests.

3. Error paths have been considered.

4. New tests no longer provide meaningful behavioral protection.

5. Remaining uncovered code is either:

   * unreachable;
   * trivial and low risk;
   * defensive code that cannot be meaningfully triggered;
   * implementation detail that should not be tested directly;
   * or otherwise not worth the maintenance cost.

6. The complete test suite has been run and every failure has been investigated. If a failure is caused by an incorrect test, correct or discard the test and document the reason. If a failure is caused by a confirmed production bug, keep the test that exposes the bug, document the bug, and report it rather than weakening or removing the test just to make the suite pass.

7. The final coverage report has been recorded.

8. At least one review has been made specifically for tests that may encode an existing bug.

Coverage is a measurement, not the objective.

A lower coverage percentage with strong behavioral tests is preferable to a higher percentage achieved through meaningless tests or tests that preserve a bug.

