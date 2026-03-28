# FastPackCloudron Accessibility (a11y) Requirements

**Date**: 2026-03-27
**Baseline score**: ~35/100 (Lighthouse Accessibility)
**Target score**: 90+ (WCAG 2.1 AA conformance)
**Scope**: `index.html` (inline CSS, ~252 lines) and `app.js` (~932 lines)

---

## Phase A: Critical -- Keyboard Users Blocked

These issues prevent keyboard-only users from operating the application at all.
Estimated total: ~45 lines of CSS, ~15 lines of HTML changes.

---

### A-1. Visible Focus Indicators on All Interactive Elements

**Problem**: No custom `:focus` styles exist. The browser default outline is suppressed
by the border-radius and background styles on buttons. Keyboard users cannot see which
element is active.

**WCAG criterion**: 2.4.7 Focus Visible (Level AA)

**Exact changes required**:

CSS (add to `<style>` block, approximately 20 lines):

```css
/*
 * A-1: Visible focus ring for all interactive elements.
 * Uses :focus-visible so mouse clicks do not show the ring.
 */
:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}

/* High-contrast focus for buttons on colored backgrounds */
#download-btn:focus-visible {
  outline-color: white;
  box-shadow: 0 0 0 4px var(--primary);
}

/* Focus ring for preview tabs */
.preview-tab:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: -2px;
  border-bottom-color: var(--primary);
}

/* Ensure remove buttons (X) have a visible focus ring */
.remove-port:focus-visible {
  outline: 2px solid var(--error);
  outline-offset: 2px;
}

/* Summary elements inside <details> */
summary:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}
```

No HTML or JS changes.

**Verification**:
1. Manual: Tab through every interactive element on the page. Every focused element
   must show a clearly visible ring.
2. Automated: `axe-core` rule `focus-visible` should pass.

**Effort**: ~20 lines of CSS.

---

### A-2. Wrap Form Inputs in `<form>` with `<fieldset>`/`<legend>` Grouping

**Problem**: The entire input area (lines 262-556 of `index.html`) is a flat `<main>`
with `<div class="form-group">` containers. There is no `<form>` element. Related
controls (radio buttons, checkbox grids, port rows) are not grouped with `<fieldset>`
and `<legend>`. Screen readers and keyboard users cannot perceive logical groupings.

**WCAG criteria**:
- 1.3.1 Info and Relationships (Level A)
- 3.3.2 Labels or Instructions (Level A)

**Exact changes required**:

HTML -- wrap the controls in `<form>` and add `<fieldset>`/`<legend>` to groups:

```html
<!-- Replace <main> opening through download button with: -->
<main>
  <form id="fastpack-form" novalidate>

    <!-- Core settings (no fieldset needed; each has its own label) -->
    <div class="form-group">
      <label for="docker-image">Docker Image</label>
      ...
    </div>

    <!-- Web Interface radio group: wrap in fieldset -->
    <fieldset class="form-group">
      <legend>Web Interface</legend>
      <div class="radio-group" role="radiogroup">
        <label><input type="radio" ...> Yes</label>
        <label><input type="radio" ...> No</label>
      </div>
    </fieldset>

    <!-- Tags checkbox grid: wrap in fieldset -->
    <!-- (inside the "Customize metadata" <details>) -->
    <fieldset class="form-group">
      <legend>Tags</legend>
      <div class="checkbox-grid" role="group" aria-label="App tags">
        ...
      </div>
    </fieldset>

    <!-- Capabilities checkbox grid: wrap in fieldset -->
    <fieldset class="form-group">
      <legend>Capabilities</legend>
      <div class="checkbox-grid" role="group" aria-label="Linux capabilities">
        ...
      </div>
    </fieldset>

    <!-- Addons checkbox grid: wrap in fieldset -->
    <fieldset class="form-group">
      <legend>Addons</legend>
      <div class="checkbox-grid" role="group" aria-label="Cloudron addons">
        ...
      </div>
    </fieldset>

    <div id="errors"></div>
    <div id="warnings"></div>

    <button type="button" id="download-btn">Download ZIP</button>
  </form>
</main>
```

CSS -- remove the default fieldset border/padding that browsers add:

```css
fieldset {
  border: none;
  padding: 0;
  margin: 0;
}

legend {
  font-weight: 600;
  margin-bottom: 0.25rem;
  padding: 0;
}
```

JS -- `downloadZip` already uses `type="button"` or `addEventListener('click', ...)`
and does not need changes. Adding `novalidate` on the form prevents browser-native
validation from conflicting with the custom validate() function.

The following groups in `index.html` need `<fieldset>/<legend>` wrapping:
1. "Web Interface" radio group (line 323-328) -- replace outer `<div class="form-group">`
   and its `<label>` with `<fieldset class="form-group">` / `<legend>`.
2. "Tags" checkbox grid (line 416-429) -- same pattern.
3. "Capabilities" checkbox grid (line 467-473) -- same pattern.
4. "Addons" checkbox grid (line 500-510) -- same pattern.
5. "Sendmail options" checkbox group (line 512-518).
6. "Database options" groups (mysql, mongodb, redis) (lines 280-293).
7. "ProxyAuth options" checkbox group (line 316-319).

**Verification**:
1. Manual: Use NVDA or VoiceOver. Navigate into each group. The screen reader must
   announce the group name (legend text) when entering the group.
2. Automated: `axe-core` rules `fieldset`, `radiogroup` should pass.

**Effort**: ~25 lines of HTML changes, ~8 lines of CSS. No JS changes.

---

## Phase B: Important -- Screen Reader Users Affected

These issues cause confusion or information loss for screen reader users.
Estimated total: ~60 lines of JS, ~15 lines of HTML.

---

### B-1. ARIA Tab Pattern for Preview Tabs

**Problem**: The preview section (lines 560-578) uses plain `<button>` elements with
`.active` class toggling. Screen readers do not announce these as tabs, do not report
which tab is selected, and do not announce the associated panel.

**WCAG criteria**:
- 4.1.2 Name, Role, Value (Level A)
- 1.3.1 Info and Relationships (Level A)

**Exact changes required**:

HTML -- add ARIA roles and attributes to the tab bar and panels:

```html
<div class="preview-tabs" role="tablist" aria-label="File preview">
  <button class="preview-tab active" role="tab"
          data-target="manifest"
          id="tab-manifest"
          aria-selected="true"
          aria-controls="panel-manifest"
          tabindex="0">manifest</button>
  <button class="preview-tab" role="tab"
          data-target="dockerfile"
          id="tab-dockerfile"
          aria-selected="false"
          aria-controls="panel-dockerfile"
          tabindex="-1">dockerfile</button>
  <!-- ... same pattern for startsh, dockerignore, readme, versions, nginx -->
</div>

<!-- Panels: -->
<div class="preview-content active" role="tabpanel"
     id="panel-manifest" data-panel="manifest"
     aria-labelledby="tab-manifest" tabindex="0">
  <pre><code id="preview-manifest"></code></pre>
</div>
<div class="preview-content" role="tabpanel"
     id="panel-dockerfile" data-panel="dockerfile"
     aria-labelledby="tab-dockerfile" tabindex="0"
     hidden>
  <pre><code id="preview-dockerfile"></code></pre>
</div>
<!-- ... same pattern for remaining panels -->
```

Key HTML rules:
- Each `<button role="tab">` gets `id`, `aria-selected`, `aria-controls`, `tabindex`.
- Each panel `<div role="tabpanel">` gets `id`, `aria-labelledby`, `tabindex="0"`.
- Inactive panels use the `hidden` attribute instead of `display: none` class.

JS -- update the tab switching logic in `app.js` (lines 900-928) to manage ARIA state
and implement arrow-key navigation per the WAI-ARIA Tabs pattern:

```js
// Inside the tab click handler, after toggling .active class:
for (const t of tabs) {
  t.setAttribute('aria-selected', t === tab ? 'true' : 'false');
  t.setAttribute('tabindex', t === tab ? '0' : '-1');
}

// Toggle panels using hidden attribute:
for (const panel of panels) {
  if (panel.dataset.panel === target) {
    panel.classList.add('active');
    panel.removeAttribute('hidden');
  } else {
    panel.classList.remove('active');
    panel.setAttribute('hidden', '');
  }
}

// Arrow key navigation on the tablist:
const tablist = document.querySelector('[role="tablist"]');
tablist.addEventListener('keydown', function (e) {
  const visibleTabs = Array.from(tabs).filter(t => t.style.display !== 'none');
  const currentIndex = visibleTabs.indexOf(document.activeElement);
  let newIndex;

  if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
    e.preventDefault();
    newIndex = (currentIndex + 1) % visibleTabs.length;
  } else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
    e.preventDefault();
    newIndex = (currentIndex - 1 + visibleTabs.length) % visibleTabs.length;
  } else if (e.key === 'Home') {
    e.preventDefault();
    newIndex = 0;
  } else if (e.key === 'End') {
    e.preventDefault();
    newIndex = visibleTabs.length - 1;
  } else {
    return;
  }

  visibleTabs[newIndex].click();
  visibleTabs[newIndex].focus();
});
```

Also update `updatePreviewNow()` in `app.js` (lines 493-497) so that the nginx tab
visibility uses `hidden` on the panel as well:

```js
// When hiding/showing nginx tab, also manage the panel hidden attribute
const nginxPanel = document.querySelector('[data-panel="nginx"]');
if (hasServices) {
  nginxTab.style.display = '';
  // Panel visibility will be handled by tab click
} else {
  nginxTab.style.display = 'none';
  if (nginxPanel) nginxPanel.setAttribute('hidden', '');
}
```

**Verification**:
1. Manual: Use NVDA/VoiceOver. Focus the tab bar. Screen reader must announce
   "manifest tab, selected, 1 of 6" or equivalent. Arrow keys must move between tabs.
   Tab key must move into the panel content.
2. Automated: `axe-core` rules `aria-required-attr`, `aria-valid-attr-value` should pass.

**Effort**: ~30 lines of HTML attribute additions, ~35 lines of JS.

---

### B-2. Live Regions for Errors and Warnings

**Problem**: The `#errors` and `#warnings` containers (lines 553-554) are populated
dynamically by JS. Screen readers do not announce these changes because there are no
`aria-live` regions.

**WCAG criteria**:
- 4.1.3 Status Messages (Level AA)
- 3.3.1 Error Identification (Level A)

**Exact changes required**:

HTML -- add `aria-live` and `role` attributes to the containers:

```html
<div id="errors" role="alert" aria-live="assertive" aria-atomic="true"></div>
<div id="warnings" role="status" aria-live="polite" aria-atomic="true"></div>
```

- `role="alert"` + `aria-live="assertive"` for errors: interrupts the user immediately.
- `role="status"` + `aria-live="polite"` for warnings: announced at next pause.
- `aria-atomic="true"` ensures the entire container is re-read, not just the diff.

JS -- no changes needed. The existing `replaceChildren()` and `appendChild()` calls
in `updatePreviewNow()` (lines 469-476) and `downloadZip()` (lines 557-568) will
trigger the live region announcements automatically.

**Verification**:
1. Manual: With NVDA/VoiceOver running, clear the Docker Image field. The error
   message "Enter a Docker image" must be announced without the user navigating to it.
2. Automated: `axe-core` rule `aria-live-region` presence check.

**Effort**: ~4 lines of HTML attribute additions. No JS changes.

---

### B-3. Accessible Labels for Dynamic "Add" and "Remove" Buttons

**Problem**: The "Add TCP port", "Add UDP port", "Add Service", "Add Task", and
"Add source" buttons (lines 522-548) use the class `.add-port-btn` but have text-only
labels that may be ambiguous. The dynamically created remove buttons (generated in
`addPortRow`, `addServiceRow`, etc. in `app.js`) display only a unicode "X" character
(`\u2715`) with no accessible name.

**WCAG criteria**:
- 4.1.2 Name, Role, Value (Level A)
- 1.3.1 Info and Relationships (Level A)

**Exact changes required**:

HTML -- add `aria-label` to the static "Add" buttons for clarity:

```html
<button type="button" class="add-port-btn" id="add-tcp-port"
        aria-label="Add TCP port mapping">Add TCP port</button>
<button type="button" class="add-port-btn" id="add-udp-port"
        aria-label="Add UDP port mapping">Add UDP port</button>
<button type="button" class="add-port-btn" id="add-scheduler-task"
        aria-label="Add scheduled task">Add Task</button>
<button type="button" class="add-port-btn" id="add-copy-from"
        aria-label="Add multi-stage copy source">Add source</button>
<button type="button" class="add-port-btn" id="add-service"
        aria-label="Add service process">Add Service</button>
```

JS -- add `aria-label` to dynamically created remove buttons. In each of these
functions, add one line after creating `removeBtn`:

In `addPortRow()` (~line 632):
```js
removeBtn.setAttribute('aria-label', `Remove ${type.toUpperCase()} port`);
```

In `addSchedulerTaskRow()` (~line 681):
```js
removeBtn.setAttribute('aria-label', 'Remove scheduled task');
```

In `addCopyFromRow()` (~line 728):
```js
removeBtn.setAttribute('aria-label', 'Remove copy source');
```

In `addServiceRow()` (~line 791):
```js
removeBtn.setAttribute('aria-label', 'Remove service');
```

Additionally, add `aria-label` attributes to the dynamically created input fields
within port/service/task rows so screen readers can distinguish them. In `addPortRow()`:

```js
nameInput.setAttribute('aria-label', `${type.toUpperCase()} port name`);
titleInput.setAttribute('aria-label', `${type.toUpperCase()} port title`);
containerInput.setAttribute('aria-label', `${type.toUpperCase()} container port number`);
defaultInput.setAttribute('aria-label', `${type.toUpperCase()} default port number`);
```

Apply equivalent labels in `addSchedulerTaskRow()`, `addCopyFromRow()`, and
`addServiceRow()`.

**Verification**:
1. Manual: Tab to each remove button. NVDA/VoiceOver must announce "Remove TCP port,
   button" (or similar), not just "times, button" or "X, button".
2. Automated: `axe-core` rule `button-name` should pass.

**Effort**: ~5 lines of HTML changes, ~20 lines of JS (aria-label setAttribute calls).

---

### B-4. Associate Field-Level Error Messages with Their Inputs

**Problem**: The `<span class="field-error" data-error-for="docker-image">` elements
(lines 265, 335, 388, 393, 550) are visually positioned near their inputs but not
programmatically associated. Screen readers will not announce the error when the user
focuses the input.

**WCAG criteria**:
- 1.3.1 Info and Relationships (Level A)
- 3.3.1 Error Identification (Level A)

**Exact changes required**:

HTML -- add `id` to each error span and `aria-describedby` to the corresponding input:

```html
<input type="text" id="docker-image" placeholder="..." aria-describedby="error-docker-image">
<span class="field-error" id="error-docker-image" data-error-for="docker-image"></span>

<input type="text" id="app-id" placeholder="..." aria-describedby="error-app-id">
<span class="field-error" id="error-app-id" data-error-for="app-id"></span>

<input type="text" id="health-check-path" value="/" aria-describedby="error-health-check-path">
<span class="field-error" id="error-health-check-path" data-error-for="health-check-path"></span>

<input type="number" id="http-port" value="8000" aria-describedby="error-http-port">
<span class="field-error" id="error-http-port" data-error-for="http-port"></span>
```

JS -- when setting field errors in `updatePreviewNow()`, also set `aria-invalid` on
the input. After line 462 in `app.js`:

```js
for (const err of result.errors) {
  const el = document.querySelector(`.field-error[data-error-for="${err.field}"]`);
  if (el) {
    el.textContent = err.message;
  }
  // Mark the input as invalid for assistive tech
  const inputEl = document.getElementById(err.field);
  if (inputEl) {
    inputEl.setAttribute('aria-invalid', 'true');
  }
}
```

And in the clearing loop (before line 460), also clear `aria-invalid`:

```js
for (const el of fieldErrors) {
  el.textContent = '';
  // Clear aria-invalid on associated input
  const fieldId = el.dataset.errorFor;
  if (fieldId) {
    const inputEl = document.getElementById(fieldId);
    if (inputEl) {
      inputEl.removeAttribute('aria-invalid');
    }
  }
}
```

**Verification**:
1. Manual: Clear the Docker Image field. Tab to it. NVDA/VoiceOver must announce
   both the label ("Docker Image") and the error ("Enter a Docker image").
2. Automated: `axe-core` rules `aria-input-field-name`, `aria-valid-attr-value`.

**Effort**: ~8 lines of HTML, ~12 lines of JS.

---

### B-5. Disable Download Button During Async ZIP Generation

**Problem**: The download button (line 556) is not disabled during the `await
zip.generateAsync()` call in `downloadZip()` (line 595). Users can click multiple
times, triggering duplicate downloads. Screen readers get no feedback that the action
is in progress.

**WCAG criteria**:
- 4.1.2 Name, Role, Value (Level A) -- button state must reflect actual state
- 2.1.1 Keyboard (Level A) -- repeated Enter on focused button triggers duplicates

**Exact changes required**:

JS -- modify `downloadZip()` in `app.js` to disable the button and provide feedback:

```js
async function downloadZip() {
  const btn = document.getElementById('download-btn');
  btn.disabled = true;
  btn.textContent = 'Generating...';
  btn.setAttribute('aria-busy', 'true');

  try {
    const config = buildConfig();
    const result = validate(config);

    const errorsContainer = document.getElementById('errors');
    errorsContainer.replaceChildren();

    if (result.errors.length > 0) {
      for (const err of result.errors) {
        const div = document.createElement('div');
        div.className = 'error';
        div.textContent = err.message;
        errorsContainer.appendChild(div);
      }
      return;                // early return; finally block re-enables
    }

    // ... existing ZIP generation code ...

    const blob = await zip.generateAsync({ type: 'blob' });
    const filename = `${sanitizeImageName(config.image) || 'cloudron-app'}-cloudron.zip`;
    saveAs(blob, filename);
  } finally {
    btn.disabled = false;
    btn.textContent = 'Download ZIP';
    btn.removeAttribute('aria-busy');
  }
}
```

CSS -- add a disabled state style:

```css
#download-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
```

**Verification**:
1. Manual: Click Download while fields are valid. Button must show "Generating..." and
   be non-clickable until the download triggers. Tab to it -- screen reader must
   announce "Generating..., button, dimmed" or equivalent.
2. Automated: No standard rule; manual test only.

**Effort**: ~12 lines of JS, ~4 lines of CSS.

---

## Phase C: Enhancement -- Visual Preferences

These issues affect user comfort and compliance with WCAG recommendations.
Estimated total: ~80 lines of CSS, ~10 lines of JS.

---

### C-1. Dark Mode via `prefers-color-scheme`

**Problem**: The app uses CSS custom properties but only defines a light theme.
Users with `prefers-color-scheme: dark` see a bright white page. No toggle exists.

**WCAG criterion**: 1.4.1 Use of Color (Level A, tangentially); not a strict WCAG
requirement but a significant usability improvement and increasingly expected.

**Exact changes required**:

CSS -- add a `prefers-color-scheme: dark` media query that redefines the custom
properties:

```css
@media (prefers-color-scheme: dark) {
  :root {
    --primary: #4da6ff;
    --primary-hover: #3d8ce0;
    --bg: #1a1a2e;
    --card-bg: #16213e;
    --text: #e0e0e0;
    --border: #3a3a5c;
    --error: #fc8181;
    --warning: #f6ad55;
    --success: #68d391;
  }

  body {
    color-scheme: dark;
  }

  input, select, textarea {
    background: var(--card-bg);
    color: var(--text);
    border-color: var(--border);
  }

  details {
    border-color: var(--border);
  }

  pre {
    background: #0d1117;
    color: #c9d1d9;
  }

  .warning {
    background: #3d2e0a;
    color: var(--warning);
  }

  .error {
    background: #2d1b1b;
    color: var(--error);
  }

  .add-port-btn {
    color: var(--text);
    border-color: var(--border);
  }

  .copy-btn {
    background: var(--card-bg);
    color: var(--text);
    border-color: var(--border);
  }

  .copy-btn:hover {
    background: var(--bg);
  }

  .preview-tab {
    color: var(--text);
  }

  .preview-tab.active {
    color: var(--primary);
  }

  footer, footer a {
    color: var(--text);
  }
}
```

No JS changes required -- the media query is automatic.

**Verification**:
1. Manual: In Chrome DevTools, toggle "prefers-color-scheme: dark" via Rendering tab.
   All backgrounds, text, inputs, buttons, and code blocks must use dark colors.
   Check that the code preview `<pre>` blocks remain legible.
2. Automated: Lighthouse "color-contrast" audit in both light and dark mode.

**Effort**: ~50 lines of CSS. No HTML or JS changes.

---

### C-2. `prefers-reduced-motion` Support

**Problem**: There are no animations in the current CSS, but the JS "Copied!" flash
(line 543 in `app.js`) uses a `setTimeout` that could be jarring. Future animations
should be preemptively gated.

**WCAG criterion**: 2.3.3 Animation from Interactions (Level AAA, but best practice)

**Exact changes required**:

CSS -- add a reduced-motion media query as a guard for any future transitions:

```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
    scroll-behavior: auto !important;
  }
}
```

JS -- optionally shorten or skip the "Copied!" flash for users who prefer reduced
motion. In `copyPreview()` (line 541):

```js
const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
const flashDuration = reduceMotion ? 0 : 1200;
if (flashDuration > 0) {
  btn.textContent = 'Copied!';
  setTimeout(function () { btn.textContent = original; }, flashDuration);
}
```

**Verification**:
1. Manual: In Chrome DevTools Rendering, enable "prefers-reduced-motion: reduce".
   No transitions or animations should play. The "Copied!" text may appear briefly
   or not at all.
2. Automated: Grep the CSS for `transition` or `animation` that are not gated by
   the media query.

**Effort**: ~8 lines of CSS, ~4 lines of JS.

---

### C-3. Fix Warning Color Contrast Ratio

**Problem**: The warning text uses `--warning: #dd6b20` on `background: #fff3cd`
(lines 231-238). This combination yields a contrast ratio of approximately 3.0:1,
which fails WCAG AA (minimum 4.5:1 for normal text).

**WCAG criterion**: 1.4.3 Contrast (Minimum) (Level AA)

**Exact changes required**:

CSS -- darken the warning text color to achieve at least 4.5:1 contrast on `#fff3cd`:

```css
/* Light mode: change --warning from #dd6b20 to #9a4e03 (contrast ~6.2:1 on #fff3cd) */
:root {
  --warning: #9a4e03;
}
```

For dark mode (from C-1), the warning values already use `#f6ad55` on `#3d2e0a`,
which yields ~7.0:1. Verify this is maintained.

Also check other color combinations:
- `--error: #e53e3e` on `#fee` = ~4.8:1 -- passes AA (barely). Consider darkening
  to `#c53030` for a more comfortable margin (~6.5:1).
- `--success: #38a169` is only used as a variable, not currently rendered. No action
  needed now but should be verified when first used.

Recommended changes:

```css
:root {
  --warning: #9a4e03;   /* was #dd6b20 -- now 6.2:1 on #fff3cd */
  --error: #c53030;     /* was #e53e3e -- now 6.5:1 on #fee */
}
```

**Verification**:
1. Manual: Use the WebAIM Contrast Checker or Chrome DevTools color picker. Each
   text-on-background combination must show >= 4.5:1.
2. Automated: Lighthouse "color-contrast" audit must pass with 0 violations.

**Effort**: ~4 lines of CSS (just changing custom property values).

---

### C-4. Hidden Content Accessibility for Conditionally Visible Sections

**Problem**: Database-specific options, SSO options, and other conditional sections
(lines 279-298, 311-320, 399-414, 512-523) use `style.display = 'none'` via the
`toggleVisibility()` function. While `display: none` does correctly remove content
from the accessibility tree, the pattern does not communicate *why* the section
appeared or provide context to screen reader users.

**WCAG criterion**: 1.3.1 Info and Relationships (Level A)

**Exact changes required**:

HTML -- add `aria-live="polite"` to the conditional option groups so screen readers
announce when they appear:

```html
<div id="mysql-options-group" style="display: none;" aria-live="polite">
<div id="mongodb-options-group" style="display: none;" aria-live="polite">
<div id="redis-options-group" style="display: none;" aria-live="polite">
<div id="postgresql-options-group" style="display: none;" aria-live="polite">
<div id="proxyauth-options-group" style="display: none;" aria-live="polite">
<div id="sendmail-options-group" style="display: none;" aria-live="polite">
<div id="scheduler-options-group" style="display: none;" aria-live="polite">
<div id="oidc-redirect-group" style="display: none;" aria-live="polite">
<div id="oidc-logout-group" style="display: none;" aria-live="polite">
<div id="oidc-token-algo-group" style="display: none;" aria-live="polite">
```

Note: Use `aria-live="polite"` (not `"assertive"`) because these are supplementary
options, not errors.

No JS changes needed -- `toggleVisibility()` already manages `display`.

**Verification**:
1. Manual: With NVDA running, change the Database dropdown from "None" to "PostgreSQL".
   The "Locale" field group must be announced politely.
2. Automated: Verify `aria-live` attribute presence on conditional groups.

**Effort**: ~10 HTML attribute additions (one per conditional group).

---

## Summary Table

| ID   | Title                             | Phase | WCAG      | Effort (lines) | Priority |
|------|-----------------------------------|-------|-----------|-----------------|----------|
| A-1  | Focus indicators                  | A     | 2.4.7 AA  | ~20 CSS         | P0       |
| A-2  | Form + fieldset/legend            | A     | 1.3.1 A   | ~25 HTML, 8 CSS | P0       |
| B-1  | ARIA tabs pattern                 | B     | 4.1.2 A   | ~30 HTML, 35 JS | P1       |
| B-2  | Live regions for errors/warnings  | B     | 4.1.3 AA  | ~4 HTML         | P1       |
| B-3  | Button aria-labels                | B     | 4.1.2 A   | ~5 HTML, 20 JS  | P1       |
| B-4  | Error-input association           | B     | 3.3.1 A   | ~8 HTML, 12 JS  | P1       |
| B-5  | Download button disabled state    | B     | 4.1.2 A   | ~12 JS, 4 CSS   | P1       |
| C-1  | Dark mode                         | C     | --        | ~50 CSS         | P2       |
| C-2  | Reduced motion                    | C     | 2.3.3 AAA | ~8 CSS, 4 JS    | P2       |
| C-3  | Warning contrast fix              | C     | 1.4.3 AA  | ~4 CSS          | P2       |
| C-4  | Conditional section live regions  | C     | 1.3.1 A   | ~10 HTML attrs  | P2       |

**Total estimated effort**: ~132 lines of CSS, ~48 lines of HTML changes, ~83 lines of JS.

---

## Testing Strategy

### Manual Testing Checklist (per requirement)

1. **Keyboard-only walkthrough**: Tab from the first input to the download button and
   through all preview tabs without using a mouse. Every interactive element must be
   reachable and visibly focused.

2. **Screen reader matrix**: Test with at least one of:
   - NVDA + Firefox (Windows)
   - VoiceOver + Safari (macOS)
   - ChromeVox + Chrome (cross-platform)

3. **Color contrast spot check**: Use Chrome DevTools color picker on every text
   element against its background in both light and dark mode.

4. **Reduced motion check**: Enable "prefers-reduced-motion: reduce" in DevTools
   Rendering and verify no animations play.

### Automated Testing

1. **axe-core** (via browser extension or `@axe-core/playwright`):
   Run against the page in default state and after triggering errors (empty Docker
   Image). Expected: 0 violations at AA level after all phases complete.

2. **Lighthouse Accessibility audit**: Target score 90+.

3. **Playwright a11y snapshot**: Add a test that captures the ARIA tree and asserts
   the tab pattern, live regions, and fieldset structure are present.

---

## Implementation Order

Recommended implementation sequence within each phase:

**Phase A** (do first -- unblocks keyboard users):
1. A-1 (focus indicators) -- pure CSS, no risk of breaking anything
2. A-2 (form structure) -- HTML restructuring, test that JS still wires correctly

**Phase B** (do second -- unblocks screen reader users):
1. B-2 (live regions) -- 4 lines, immediate high impact
2. B-4 (error-input association) -- pairs well with B-2
3. B-3 (button labels) -- straightforward aria-label additions
4. B-1 (ARIA tabs) -- most complex item, do last in phase
5. B-5 (download button state) -- independent, can be done in parallel

**Phase C** (do third -- polish):
1. C-3 (contrast fix) -- 4 lines, instant win
2. C-2 (reduced motion) -- small, future-proofs the codebase
3. C-4 (conditional section live regions) -- small, complements B-2
4. C-1 (dark mode) -- largest item, purely additive CSS
