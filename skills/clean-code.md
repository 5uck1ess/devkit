---
name: devkit:clean-code
description: Clean code principles — meaningful names, small functions, single responsibility, stepdown rule, flat nesting.
---

# Clean Code

## Meaningful Names

- Names should reveal intent. `getUserById` not `getData`. `isExpired` not `check`.
- Don't abbreviate unless the abbreviation is universally understood (`url`, `id`, `config`).
- Booleans read as questions: `isReady`, `hasPermission`, `shouldRetry`.
- Functions read as actions: `calculateTotal`, `validateInput`, `sendNotification`.

## Small Functions

A function should do one thing. If you're describing what a function does and use the word "and," it does too much.

Aim for 5-15 lines. Not a hard rule, but long functions almost always contain extractable sub-functions.

## Single Responsibility

Every module/class/function should have one reason to change. If a module handles both parsing and validation, a change to parsing rules forces you to touch validation code (and vice versa).

Ask: "If requirement X changes, how many files do I touch?" If the answer is many, responsibilities are entangled.

## The Stepdown Rule

Organize code so readers encounter high-level logic first, details later. A file should read like a newspaper article — headline, summary, then details.

```
// Good: high-level flow is immediately clear
function processOrder(order) {
  validate(order);
  const total = calculateTotal(order.items);
  return submitPayment(order.customer, total);
}

// Supporting functions follow below
function validate(order) { /* ... */ }
function calculateTotal(items) { /* ... */ }
function submitPayment(customer, total) { /* ... */ }
```

Public/exported functions at the top. Private/helper functions below.

## Flat Nesting

Deeply nested code is hard to follow. Prefer early returns, guard clauses, and extraction.

```
// Bad: nested
function process(input) {
  if (input) {
    if (input.isValid) {
      if (!input.isProcessed) {
        return doWork(input);
      }
    }
  }
  return null;
}

// Good: flat
function process(input) {
  if (!input) return null;
  if (!input.isValid) return null;
  if (input.isProcessed) return null;
  return doWork(input);
}
```

If you're indented more than 2 levels, look for an extraction or early return.
