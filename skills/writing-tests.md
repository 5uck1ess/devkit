---
name: devkit:writing-tests
description: Test writing guide — one assertion per behavior, test contracts not implementation, arrange-act-assert, mocking strategy.
---

# Writing Tests

## Arrange-Act-Assert

Every test has three parts:

```
// Arrange — set up the preconditions
const cart = new Cart();
cart.addItem({ name: "Widget", price: 10 });

// Act — do the thing being tested
const total = cart.calculateTotal();

// Assert — verify the result
expect(total).toBe(10);
```

Keep each section short. If arrange is 20 lines, extract a helper. If you need multiple act+assert blocks, write multiple tests.

## One Behavior Per Test

Each test verifies one behavior. Not one function — one *behavior*.

Good: each test checks one thing
```
test("rejects empty username", () => { /* ... */ });
test("rejects username with spaces", () => { /* ... */ });
test("accepts valid username", () => { /* ... */ });
```

Bad: multiple behaviors in one test
```
test("validates username", () => {
  expect(validate("")).toBe(false);
  expect(validate("has space")).toBe(false);
  expect(validate("valid")).toBe(true);
});
```

When a test fails, you should know exactly what broke from the test name alone.

## Test the Contract, Not the Implementation

Tests should verify *what* a function does, not *how* it does it.

- Assert on return values, side effects, and state changes.
- Don't assert on internal method calls, private state, or execution order (unless order is part of the contract).
- If you refactor internals and tests break, those tests were testing implementation.

## When to Mock

**Mock these:**
- External services (APIs, databases, file system)
- Non-deterministic things (time, randomness)
- Slow dependencies that would make tests impractical

**Don't mock these:**
- The thing you're testing
- Simple value objects or pure functions
- Things that are fast and deterministic

Prefer real dependencies when practical. Mocks test that you call things correctly; real dependencies test that things actually work.

## Test Names

Use names that describe the scenario and expected outcome:

```
"returns empty array when no items match filter"
"throws when called with negative amount"
"sends notification email after order is placed"
```

If you can't name the test clearly, you may not understand the requirement yet.
