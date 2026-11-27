/* hbt-ui.js: shared UI layer for every Happy Baby Tracker page.
   1. shadcn-svelte form controls (Input, Textarea, Select, Checkbox, RadioGroup) as styled native elements.
   2. ToggleGroup chips (.hbt-chip): one pattern for Timeline filters and Export modules.
   3. Drawer-first forms: a link with data-drawer="Title" opens its own page (?embed=1) in a bottom Drawer
      over the current page. The same link still works as a normal link (no JS, new tab, modified click).
   4. Age line helper for the title subline (HBTUI.age).
   Prototype note: the Drawer hosts the real form page in an iframe so the form has one implementation.
   In production this is a Drawer component rendering the same form component as the route. */
(function () {
  'use strict';
  var EMBED = /[?&]embed=1/.test(location.search || '');
  var CHECK = "url(\"data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='white' stroke-width='3.2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M20 6 9 17l-5-5'/%3E%3C/svg%3E\")";
  var CHECK_P = "url(\"data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='%23b4380a' stroke-width='3' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M20 6 9 17l-5-5'/%3E%3C/svg%3E\")";
  var CHEV = "url(\"data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='%23857b73' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='m6 9 6 6 6-6'/%3E%3C/svg%3E\")";
  var RING = '0 0 0 3px color-mix(in srgb,var(--primary) 32%,transparent)';
  var TXT = 'input[type=text],input[type=number],input[type=date],input[type=time],input[type=datetime-local],input[type=email],input[type=search],input[type=tel],input[type=url],input[type=password],input:not([type])';
  var css = [
    /* Input / Textarea / Select */
    TXT + ',select,textarea{box-sizing:border-box;min-height:48px;border:1px solid var(--border)!important;border-radius:8px!important;background-color:var(--bg)!important;color:var(--fg);padding:0 12px;font-size:16px;line-height:1.4;box-shadow:0 1px 2px rgb(0 0 0/.05);transition:box-shadow .12s,border-color .12s}',
    'textarea{padding:10px 12px;min-height:96px;line-height:1.5}',
    'select{-webkit-appearance:none;appearance:none;padding-right:36px;background-image:' + CHEV + '!important;background-repeat:no-repeat!important;background-position:right 12px center!important;background-size:16px!important}',
    TXT.split(',').map(function (s) { return s + ':focus-visible'; }).join(',') + ',select:focus-visible,textarea:focus-visible{outline:none;border-color:var(--primary)!important;box-shadow:' + RING + '}',
    '[aria-invalid=true]{border-color:var(--destructive)!important}',
    /* Page column: list and detail pages are 720 px wide at most, left-aligned; full-page forms set their own 448 px column inline. */
    '[role=main]>*:not([style*="max-width"]){max-width:720px;width:100%;box-sizing:border-box}',
    'input:disabled,select:disabled,textarea:disabled{opacity:.5;cursor:not-allowed}',
    /* Checkbox */
    'input[type=checkbox]:not([role=switch]){-webkit-appearance:none;appearance:none;flex:none;box-sizing:border-box;width:20px!important;height:20px!important;min-height:0!important;margin:0;padding:0;border:1px solid color-mix(in srgb,var(--fg) 55%,var(--bg))!important;border-radius:4px!important;background-color:var(--bg)!important;background-repeat:no-repeat;background-position:center;background-size:14px;cursor:pointer;box-shadow:0 1px 2px rgb(0 0 0/.05)}',
    'input[type=checkbox]:not([role=switch]):checked{background-color:var(--primary)!important;border-color:var(--primary)!important;background-image:' + CHECK + '}',
    'input[type=checkbox]:not([role=switch]):focus-visible,input[type=radio]:focus-visible{outline:none;box-shadow:' + RING + '}',
    /* RadioGroup */
    'input[type=radio]{-webkit-appearance:none;appearance:none;flex:none;box-sizing:border-box;width:20px!important;height:20px!important;min-height:0!important;margin:0;padding:0;border:1px solid color-mix(in srgb,var(--fg) 55%,var(--bg))!important;border-radius:50%!important;background-color:var(--bg)!important;cursor:pointer;box-shadow:0 1px 2px rgb(0 0 0/.05)}',
    'input[type=radio]:checked{border-color:var(--primary)!important;background-image:radial-gradient(circle,var(--primary) 0 4.5px,transparent 5.5px)}',
    /* ToggleGroup chip: <button aria-pressed> or <label class="hbt-chip"><input type=checkbox>text</label> */
    '.hbt-chip{position:relative;box-sizing:border-box;display:inline-flex;align-items:center;gap:6px;min-height:44px;padding:0 14px;border-radius:8px;border:1px solid var(--border);background:var(--card);color:var(--fg);font-size:14px;font-weight:500;line-height:1;cursor:pointer;text-decoration:none}',
    '.hbt-chip[aria-pressed=true],.hbt-chip:has(input:checked){background:var(--soft);border-color:var(--primary);font-weight:600}',
    '.hbt-chip[aria-pressed=true]::before,.hbt-chip:has(input:checked)::before{content:"";flex:none;width:14px;height:14px;background:' + CHECK_P + ' center/14px no-repeat;order:-1}',
    '.hbt-chip input[type=checkbox]{position:absolute!important;inset:0;width:100%!important;height:100%!important;opacity:0;margin:0;border:0!important;cursor:pointer}',
    '.hbt-chip:has(input:focus-visible),.hbt-chip:focus-visible{outline:none;box-shadow:' + RING + '}',
    /* RadioGroup row (Change theme): label card, radio on the left, preview swatch on the right */
    '.hbt-radiorow{position:relative;box-sizing:border-box;display:flex;align-items:center;gap:12px;min-height:60px;padding:8px 12px;border-radius:12px;border:1px solid var(--border);background:var(--card);cursor:pointer}',
    '.hbt-radiorow:has(input:checked){border-color:var(--primary);background:var(--soft)}',
    '.hbt-radiorow:has(input:focus-visible){box-shadow:' + RING + '}',
    /* Drawer */
    '.hbt-dr{position:absolute;inset:0;z-index:30;display:flex;align-items:flex-end;justify-content:center;background:rgba(20,12,8,.5)}',
    '.hbt-dr-p{box-sizing:border-box;width:100%;height:90%;display:flex;flex-direction:column;background:var(--bg);color:var(--fg);border-radius:16px 16px 0 0;border-top:1px solid var(--border);box-shadow:0 -8px 30px rgb(0 0 0/.2)}',
    '.hbt-dr-h{flex:none;display:flex;align-items:center;gap:8px;padding:8px 8px 0 16px}',
    '.hbt-dr-grab{position:absolute;left:50%;top:6px;width:36px;height:4px;margin-left:-18px;border-radius:2px;background:var(--border)}',
    '.hbt-dr-t{flex:1;min-width:0;margin:12px 0 0;font-size:18px;font-weight:600}',
    '.hbt-dr-x{flex:none;width:44px;height:44px;margin-top:8px;border:0;border-radius:8px;background:transparent;color:var(--fg);cursor:pointer;display:inline-flex;align-items:center;justify-content:center}',
    '.hbt-dr-x:focus-visible{outline:none;box-shadow:' + RING + '}',
    '.hbt-dr-f{flex:1;min-height:0;width:100%;border:0;background:transparent}',
    '.hbt-dr-fit{height:auto;max-height:90%;overflow:auto}.hbt-dr-fit .hbt-dr-f{min-height:120px}',
    '.hbt-toast{position:absolute;left:12px;right:12px;bottom:84px;z-index:40;padding:12px 16px;border-radius:12px;background:var(--fg);color:var(--bg);font-size:14px;line-height:1.4}',
    /* Buttons: the 48 px minimum height includes the border (primary and secondary are the same height) */
    'button,a[data-shadcn=Button],a[style*="min-height"],summary[style*="min-height"]{box-sizing:border-box}',
    /* Disclosure chevron: <svg data-chev> as the last child of <summary> turns when the <details> opens */
    'details>summary>[data-chev]{transition:none}details[open]>summary>[data-chev]{transform:rotate(180deg)}',
    ':where(a,button,summary):focus-visible{outline:2px solid var(--primary);outline-offset:2px}',
    '@media (prefers-reduced-motion:no-preference){details>summary>[data-chev]{transition:transform .15s ease-out}.hbt-dr-p{animation:hbtUp .2s ease-out}@keyframes hbtUp{from{transform:translateY(24px);opacity:.6}to{transform:none;opacity:1}}}'
  ];
  /* wide-layout:v1 */
  css.push(
    '[role=main]>[data-w=grid]>*:not([style*="max-width"]),[role=main]>[data-w=grid]>[data-w]>*:not([style*="max-width"]){max-width:720px;width:100%;box-sizing:border-box}',
    '[data-w]:not([style]){display:contents}',
    'html[data-wide="1"] [data-hbt=app]{width:1280px!important}',
    'html[data-wide="1"] [role=main]:has([data-w])>*{width:min(100%,920px)!important;max-width:920px!important;margin-inline:auto!important;box-sizing:border-box}',
    'html[data-wide="1"] [role=main]:has([data-w])>[data-shadcn=Card],html[data-wide="1"] [role=main]:has([data-w])>[data-shadcn=Tabs]{width:448px!important;max-width:448px!important;margin-left:max(0px,calc((100% - 920px)/2))!important;margin-right:auto!important}',
    'html[data-wide="1"] [data-w=grid]{display:grid!important;grid-template-columns:448px 448px;column-gap:24px!important;row-gap:18px!important;align-items:start;width:920px!important;max-width:920px!important}',
    'html[data-wide="1"] [data-w=grid]>*{grid-column:1/-1;min-width:0}',
    'html[data-wide="1"] [data-w=grid]>*:not([data-w]){max-width:none!important}',
    'html[data-wide="1"] [data-w=grid]>[data-w=l]{grid-column:1;max-width:448px!important}',
    'html[data-wide="1"] [data-w=grid]>[data-w=r]{grid-column:2;max-width:448px!important}',
    'html[data-wide="1"] [data-w=l]:not([style]),html[data-wide="1"] [data-w=r]:not([style]){display:flex;flex-direction:column;gap:18px;width:100%}',
    'html[data-wide="1"] [data-w=form]{max-width:920px!important}',
    'html[data-wide="1"] [data-w=form] [data-shadcn=Alert],html[data-wide="1"] [data-w=form] [data-shadcn=Collapsible],html[data-wide="1"] [data-w=form] [data-shadcn=Card]{max-width:448px}',
    'html[data-wide="1"] [data-w=grid2]{display:grid!important;grid-template-columns:448px 448px;grid-template-rows:auto 1fr;column-gap:24px!important;row-gap:14px!important;align-items:start}',
    'html[data-wide="1"] [data-w=fa]:not([style]),html[data-wide="1"] [data-w=fb]:not([style]){display:flex;flex-direction:column;gap:14px;min-width:0}',
    'html[data-wide="1"] [data-w=grid2][style*="gap:16px"]>[data-w=fa],html[data-wide="1"] [data-w=grid2][style*="gap:16px"]>[data-w=fb]{gap:16px}',
    'html[data-wide="1"] [data-w=fa]{grid-column:1;grid-row:1}',
    'html[data-wide="1"] [data-w=fb]{grid-column:2;grid-row:1/span 2}',
    'html[data-wide="1"] [data-w=ff]{grid-column:1;grid-row:2}');
  if (EMBED) {
    css.push('[data-hbt=outer]{padding:0!important;gap:0!important;background:transparent!important;min-height:0!important;display:block!important}',
      '[data-hbt=panel],[data-hbt=head],[data-hbt=tabs],[data-hbt=side],h1,[data-sub]{display:none!important}',
      '[data-hbt=app]{width:100%!important;height:100vh!important;border-radius:0!important;box-shadow:none!important;border:0!important}',
      /[?&]fit=1/.test(location.search || '') ? '[data-hbt=app]{height:auto!important;min-height:0!important}[data-hbt=frame]{overflow:visible!important;min-height:0!important}[data-hbt=frame]>div{overflow:visible!important;flex:none!important;min-height:0!important}' : '',
      'html,body{background:transparent}');
  }
  var st = document.createElement('style');
  st.id = 'hbt-ui-css';
  st.textContent = css.join('\n');
  (document.head || document.documentElement).appendChild(st);


  /* prototype option: "≥1280 two-column" next to Phone and ≥768 side-nav (sets html[data-wide=1]; ?wide=1 starts in it) */
  if (!EMBED) {
    var de = document.documentElement;
    if (/[?&]wide=1/.test(location.search || '')) de.setAttribute('data-wide', '1');
    var wideBtn = function (txt) {
      var bs = document.querySelectorAll('[data-hbt=panel] button');
      for (var i = 0; i < bs.length; i++) if (bs[i].textContent.trim() === txt) return bs[i];
      return null;
    };
    var wideBusy = false;
    var seg = function (on) { return 'flex:1;min-height:34px;border-radius:8px;border:1px solid ' + (on ? '#2a2420' : '#d8d0c8') + ';background:' + (on ? '#2a2420' : '#fff') + ';color:' + (on ? '#fff' : '#2a2420') + ';font-size:12.5px;font-weight:600;cursor:pointer;padding:0 6px'; };
    var setSt = function (el, v) { if (el.getAttribute('style') !== v) el.setAttribute('style', v); };
    var syncWide = function () {
      var side = wideBtn('≥768 side-nav'), phone = wideBtn('Phone');
      if (!side || !phone) return;
      var on = de.getAttribute('data-wide') === '1', w = document.querySelector('[data-hbt=panel] [data-wide-btn]');
      if (!w) {
        w = side.cloneNode(true); w.setAttribute('data-wide-btn', '1'); w.textContent = '≥1280 two-column';
        w.addEventListener('click', function () {
          de.setAttribute('data-wide', '1');
          var sd = wideBtn('≥768 side-nav');
          if (sd && sd.getAttribute('style').indexOf('background:#2a2420') < 0) { wideBusy = true; sd.click(); wideBusy = false; }
          setTimeout(syncWide, 0);
        });
        side.parentNode.style.flexWrap = 'wrap';
        side.parentNode.appendChild(w);
      }
      side.parentNode.style.flexWrap = 'wrap';
      setSt(w, seg(on) + ';flex:1 1 100%');
      if (on) setSt(side, seg(false));
    };
    document.addEventListener('click', function (e) {
      var b = e.target && e.target.closest ? e.target.closest('[data-hbt=panel] button') : null;
      if (!b || wideBusy || b.hasAttribute('data-wide-btn')) return;
      var t = b.textContent.trim();
      if (t === 'Phone' || t === '≥768 side-nav') { de.removeAttribute('data-wide'); setTimeout(syncWide, 0); }
    }, true);
    var wmo = new MutationObserver(function () { syncWide(); });
    var wstart = function () { wmo.observe(document.body, { childList: true, subtree: true }); syncWide(); };
    if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', wstart); else wstart();
  }

  /* ---------- age line ---------- */
  function plural(n, w) { return n + ' ' + w + (n === 1 ? '' : 's'); }
  function fmt(iso) {
    var b = iso.split('-').map(Number), t = new Date();
    var y = t.getFullYear(), mo = t.getMonth() + 1, d = t.getDate();
    var months = (y - b[0]) * 12 + (mo - b[1]), days = d - b[2];
    if (days < 0) { months -= 1; days += new Date(y, mo - 1, 0).getDate(); }
    if (months < 0) return '';
    var yy = Math.floor(months / 12), mm = months % 12, o = [];
    if (yy) o.push(plural(yy, 'year'));
    if (mm) o.push(plural(mm, 'month'));
    if (months < 12) o.push(plural(days, 'day'));
    return o.join(' ') || '0 days';
  }
  function age(kidId) {
    try {
      var H = window.HBT, m = H.kidMeta(kidId || H.kid());
      return m && m.born ? fmt(m.born) : '';
    } catch (e) { return ''; }
  }
  /* corrected age text for a baby born before 37 weeks (empty otherwise or from 2 years) */
  function corrected(kidId) {
    try {
      var H = window.HBT, m = H.kidMeta(kidId || H.kid());
      if (!m || !m.born || !(m.gaw < 37)) return '';
      var b = new Date(m.born + 'T12:00:00'); b.setDate(b.getDate() + (40 - m.gaw) * 7 - (m.gad || 0));
      var iso = b.getFullYear() + '-' + ('0' + (b.getMonth() + 1)).slice(-2) + '-' + ('0' + b.getDate()).slice(-2);
      return fmt(iso);
    } catch (e) { return ''; }
  }
  /* header data shared by every page: current child + where the switcher returns to */
  function hdr(page) {
    var o = { name: '', initial: '', href: 'Choose Child.dc.html' };
    try {
      var H = window.HBT, id = H.kid(), nm = H.kidName(id) || '';
      var p = page || decodeURIComponent((location.pathname.split('/').pop() || 'Kid Home.dc.html'));
      o.name = nm || 'Choose a child';
      o.initial = nm ? Array.from(nm.trim())[0].toUpperCase() : '+';
      o.href = 'Choose Child.dc.html?from=' + encodeURIComponent(p);
    } catch (e) {}
    return o;
  }

  /* ---------- theme: one shared preference (Light | Dark | Auto), stored with the other prefs ----------
     Every page derives its palette from HBTUI.isDark(this); Auto follows prefers-color-scheme live.
     Pages subscribe once (first render) so a change made anywhere re-renders them without a reload. */
  var subs = [];
  var mqd = window.matchMedia ? window.matchMedia('(prefers-color-scheme: dark)') : null;
  function themePref() {
    try { var v = window.HBT.pref('theme'); return v === 'light' || v === 'dark' ? v : 'auto'; } catch (e) { return 'auto'; }
  }
  function resolveDark() { var t = themePref(); return t === 'dark' || (t === 'auto' && !!(mqd && mqd.matches)); }
  function applyScheme() { try { document.documentElement.style.colorScheme = resolveDark() ? 'dark' : 'light'; } catch (e) {} }
  function notify() { applyScheme(); subs.slice().forEach(function (f) { try { f(); } catch (e) {} }); }
  function isDark(self) {
    if (self && !self.__hbtTheme) { self.__hbtTheme = 1; subs.push(function () { self.forceUpdate(); }); }
    applyScheme();
    return resolveDark();
  }
  function setTheme(v) {
    if (v !== 'light' && v !== 'dark') v = 'auto';
    try { window.HBT.setPref('theme', v); } catch (e) {}
    notify();
  }
  if (mqd) { if (mqd.addEventListener) mqd.addEventListener('change', notify); else if (mqd.addListener) mqd.addListener(notify); }
  window.addEventListener('storage', notify);
  var THEMES = [
    ['light', 'Light', 'Bright background.', 'background:#fbf7f2;border:1px solid #cfc6bc'],
    ['dark', 'Dark', 'Dimmed background, easier at night.', 'background:#2a2420;border:1px solid #6b625a'],
    ['auto', 'Auto', 'Follows your device setting.', 'background:linear-gradient(135deg,#fbf7f2 50%,#2a2420 50%);border:1px solid #8a8178']];
  function themeLabel() { return { light: 'Light', dark: 'Dark', auto: 'Auto' }[themePref()]; }
  /* template values for the Change theme sheet: isTheme + a RadioGroup of three options */
  function themeVals(sheet) {
    var cur = themePref();
    return {
      isTheme: sheet === 'theme',
      themeOpts: THEMES.map(function (t) {
        return { id: t[0], label: t[1], hint: t[2], swatch: t[3] + ';width:32px;height:32px;border-radius:8px;flex:none', checked: cur === t[0], on: function () { setTheme(t[0]); } };
      })
    };
  }

  /* ---------- navigation model: one list for the phone More sheet and the desktop side nav ----------
     Phone: Home / Timeline / Growth / More tabs; More opens a sheet with these entries.
     >=768: the side nav shows the same entries inline (More is dropped), so each is one click away.
     Entries that open a sheet are handled in place by the page that owns the sheet (o.sheets), otherwise
     they link to Kid Home with ?open=<sheet>. */
  var NAVIC = {
    vaccines: '<path d="m18 2 4 4"/><path d="m17 7 3-3"/><path d="M19 9 8.7 19.3c-1 1-2.5 1-3.4 0l-.6-.6c-1-1-1-2.5 0-3.4L15 5"/><path d="m9 11 4 4"/><path d="m5 19-3 3"/><path d="m14 4 6 6"/>',
    medication: '<path d="M13.4 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-7.4"/><path d="M2 6h4"/><path d="M2 10h4"/><path d="M2 14h4"/><path d="M2 18h4"/><path d="M21.378 5.626a1 1 0 1 0-3.004-3.004l-5.01 5.012a2 2 0 0 0-.506.854l-.837 2.87a.5.5 0 0 0 .62.62l2.87-.837a2 2 0 0 0 .854-.506z"/>',
    milestones: '<path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z"/><line x1="4" x2="4" y1="22" y2="15"/>',
    export: '<path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/><path d="M14 2v4a2 2 0 0 0 2 2h4"/><path d="M12 18v-6"/><path d="m9 15 3 3 3-3"/>',
    modules: '<rect width="7" height="7" x="3" y="3" rx="1"/><rect width="7" height="7" x="14" y="3" rx="1"/><rect width="7" height="7" x="14" y="14" rx="1"/><rect width="7" height="7" x="3" y="14" rx="1"/>',
    child: '<path d="M9 12h.01"/><path d="M15 12h.01"/><path d="M10 16c.5.3 1.2.5 2 .5s1.5-.2 2-.5"/><path d="M19 6.3a9 9 0 0 1 1.8 3.9 2 2 0 0 1 0 3.6 9 9 0 0 1-17.6 0 2 2 0 0 1 0-3.6A9 9 0 0 1 12 3c2 0 3.5 1.1 3.5 2.5s-.9 2.5-2 2.5c-.8 0-1.5-.4-1.5-1"/>',
    settings: '<path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/>',
    me: '<path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>',
    logout: '<path d="m16 17 5-5-5-5"/><path d="M21 12H9"/><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>',
    theme: '<path d="M12 8a2.83 2.83 0 0 0 4 4 4 4 0 1 1-4-4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.9 4.9 1.4 1.4"/><path d="m17.7 17.7 1.4 1.4"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.3 17.7-1.4 1.4"/><path d="m19.1 4.9-1.4 1.4"/>'
  };
  var PARENT = { 'Course.dc.html': 'Medication.dc.html', 'Dose.dc.html': 'Medication.dc.html', 'Milestone Form.dc.html': 'Milestones.dc.html', 'Add a Child.dc.html': 'Choose Child.dc.html', 'Child Settings.dc.html': 'Child Profile.dc.html', 'Access.dc.html': 'Child Profile.dc.html' };
  function curPage() { try { return decodeURIComponent(location.pathname.split('/').pop() || 'Kid Home.dc.html'); } catch (e) { return ''; } }
  var SIDE_BASE = 'display:flex;align-items:center;gap:10px;min-height:44px;padding:0 10px;border-radius:10px;font-size:14px;font-weight:600;text-decoration:none;';
  /* o: {cg:boolean, hide:['medication'], sheets:['theme',...], open:function(sheetId)} */
  function nav(o) {
    o = o || {};
    var cur = curPage(); cur = PARENT[cur] || cur;
    var hide = o.hide || [], sheets = o.sheets || ['theme'];
    var DEF = [
      ['vaccines', 'Vaccines', 'Open', 'Vaccines.dc.html', 0],
      ['medication', 'Medication', 'Open', 'Medication.dc.html', 0],
      ['milestones', 'Milestones', 'Open', 'Milestones.dc.html', 0],
      ['export', 'Export PDF', 'Open', 'Export.dc.html', 0],
      ['modules', 'Modules', 'Guardian', '#modules', 1],
      ['child', 'Child profile', 'Guardian', 'Child Profile.dc.html', 1],
      ['settings', 'Settings', 'Guardian', '#settings', 1],
      ['me', 'My profile', 'Open', 'My Profile.dc.html', 0],
      ['theme', 'Change theme', themeLabel(), '#theme', 0],
      ['logout', 'Sign out', 'This device', 'Sign In.dc.html', 0]];
    var more = [], side = [];
    DEF.forEach(function (d) {
      if (d[4] && o.cg) return;
      if (hide.indexOf(d[0]) >= 0) return;
      var isSheet = d[3].charAt(0) === '#', sid = d[3].slice(1), inPlace = isSheet && sheets.indexOf(sid) >= 0 && typeof o.open === 'function';
      var href = !isSheet ? d[3] : inPlace ? d[3] : 'Kid Home.dc.html?open=' + sid;
      var on = inPlace ? function (e) { if (e && e.preventDefault) e.preventDefault(); o.open(sid); } : undefined;
      var here = !isSheet && d[3] === cur;
      more.push({ label: d[1], note: d[2], href: href, on: on });
      side.push({ label: d[1], href: href, on: on, icon: { __html: NAVIC[d[0]] }, cur: here ? 'page' : undefined,
        sideStyle: SIDE_BASE + 'background:' + (here ? 'var(--bg)' : 'transparent') + ';color:' + (here ? 'var(--primary)' : 'var(--fg)') });
    });
    /* hairline between the three primary destinations and the entries that used to sit behind More */
    if (side.length) side[0].sideStyle += ';margin-top:12px;box-shadow:0 -6px 0 -5px var(--border)';
    /* Sign out is the last entry in both lists (legacy: a POST to /logout that clears the local session even when the API is down) */
    return { more: more, side: side };
  }


  /* ---------- vaccine sample records: ONE England schedule table + the stored child's sample records ----------
     Vaccines (the page) and Home (Due rows) both read this, so they cannot disagree. Rows:
     [visit, from (months), group, antigens, dose, until (months)]. Age always counts from the stored birth date. */
  var VUK = [['8 weeks', 1.85, '6-in-1', ['Diphtheria', 'Tetanus', 'Pertussis', 'Polio', 'Hib', 'Hepatitis B'], 1, 2.85], ['8 weeks', 1.85, 'Rotavirus', ['Rotavirus'], 1, 3.8], ['8 weeks', 1.85, 'MenB', ['Meningococcal B'], 1, 2.85],
    ['12 weeks', 2.77, '6-in-1', ['Diphtheria', 'Tetanus', 'Pertussis', 'Polio', 'Hib', 'Hepatitis B'], 2, 3.77], ['12 weeks', 2.77, 'Pneumococcal', ['Pneumococcal'], 1, 3.77], ['12 weeks', 2.77, 'Rotavirus', ['Rotavirus'], 2, 3.8],
    ['16 weeks', 3.69, '6-in-1', ['Diphtheria', 'Tetanus', 'Pertussis', 'Polio', 'Hib', 'Hepatitis B'], 3, 4.69], ['16 weeks', 3.69, 'MenB', ['Meningococcal B'], 2, 4.69],
    ['1 year', 12, 'MMR', ['Measles', 'Mumps', 'Rubella'], 1, 13], ['1 year', 12, 'Pneumococcal', ['Pneumococcal'], 2, 13], ['1 year', 12, 'MenB', ['Meningococcal B'], 3, 13],
    ['3 years 4 months', 40, 'MMR', ['Measles', 'Mumps', 'Rubella'], 2, 41], ['3 years 4 months', 40, '4-in-1 booster', ['Diphtheria', 'Tetanus', 'Pertussis', 'Polio'], 1, 41]];
  function ageMonths(kidId, now) {
    try { var H = window.HBT, m = H.kidMeta(kidId || H.kid()); return ((now || Date.now()) - Date.parse(m.born + 'T12:00:00')) / (30.44 * 864e5); } catch (e) { return 0; }
  }
  /* sample records: every dose whose window has ended was given, except Rotavirus dose 2 (left open so the sample shows an overdue row) */
  function vaccRows(kidId, now) {
    var a = ageMonths(kidId, now);
    return VUK.map(function (r, i) {
      var id = 'uk' + i, given = r[5] < a && id !== 'uk5', st = given ? 'given' : a < r[1] ? 'upcoming' : a < r[5] ? 'due' : 'overdue';
      return { id: id, visit: r[0], from: r[1], group: r[2], dose: r[4], until: r[5], given: given, status: st };
    });
  }
  function vaccDue(kidId, now) {
    var rows = vaccRows(kidId, now).filter(function (r) { return r.status === 'due' || r.status === 'overdue'; });
    rows.sort(function (x, y) { return (x.status === 'overdue' ? 0 : 1) - (y.status === 'overdue' ? 0 : 1) || x.from - y.from; });
    return rows.map(function (r) { return { title: r.group + ', dose ' + r.dose, tag: r.status === 'overdue' ? 'Overdue' : 'Due' }; });
  }

  window.HBTUI = { age: age, ageFrom: fmt, corrected: corrected, hdr: hdr, embed: EMBED, isDark: isDark, theme: themePref, setTheme: setTheme, themeLabel: themeLabel, themeVals: themeVals, nav: nav, curPage: curPage, vaccUK: VUK, ageMonths: ageMonths, vaccRows: vaccRows, vaccDue: vaccDue };

  function frame() { return document.querySelector('[data-hbt=frame]'); }
  function once(fn) { var done = false; return function () { if (!done) { done = true; fn.apply(null, arguments); } }; }

  /* ---------- embedded form page (inside the Drawer) ---------- */
  if (EMBED) {
    var post = function (saved) { try { parent.postMessage({ hbt: 'close', saved: saved || '' }, '*'); } catch (e) {} };
    document.addEventListener('click', function (e) {
      var a = e.target.closest && e.target.closest('a[href]');
      if (!a) return;
      var h = a.getAttribute('href') || '';
      if (a.hasAttribute('data-top')) { e.preventDefault(); try { parent.postMessage({ hbt: 'close', go: h }, '*'); } catch (x) {} return; }
      if (a.hasAttribute('data-switch')) { e.preventDefault(); try { parent.postMessage({ hbt: 'close', reload: true }, '*'); } catch (x) {} return; }
      if (/\.dc\.html/.test(h)) { e.preventDefault(); post(''); }
    }, true);
    document.addEventListener('keydown', function (e) { if (e.key === 'Escape') post(''); });
    var sent = once(post);
    var mo = new MutationObserver(function () {
      var els = document.querySelectorAll('[role=status]');
      for (var i = 0; i < els.length; i++) {
        var n = els[i].firstChild, t = n ? (n.nodeType === 3 || (n.nodeType === 1 && n.tagName === 'SPAN') ? n.textContent.trim() : '') : '';
        if (t) { sent(t); return; }
      }
    });
    var start = function () {
      mo.observe(document.body, { childList: true, subtree: true, characterData: true });
      var q = new URLSearchParams(location.search);
      if (q.get('fit') === '1') {
        var sz = function () {
          var ap = document.querySelector('[data-hbt=app]');
          if (ap) try { parent.postMessage({ hbt: 'size', h: Math.ceil(ap.getBoundingClientRect().height) }, '*'); } catch (x) {}
        };
        if (window.ResizeObserver) new ResizeObserver(sz).observe(document.body);
        setInterval(sz, 400);
      }
      var click = function (txt) {
        var bs = document.querySelectorAll('[data-hbt=panel] button');
        for (var i = 0; i < bs.length; i++) if (bs[i].textContent.trim() === txt) { bs[i].click(); return; }
      };
      setTimeout(function () { if (q.get('cg') === '1') click('Caregiver'); }, 120);
    };
    if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', start); else start();
    return;
  }

  /* ---------- Drawer launcher (list page) ---------- */
  var cur = null;
  function closeDrawer(restore) {
    if (!cur) return;
    var t = cur.trigger;
    cur.el.parentNode && cur.el.parentNode.removeChild(cur.el);
    document.removeEventListener('keydown', cur.key, true);
    cur = null;
    if (restore !== false && t && t.focus) try { t.focus(); } catch (e) {}
  }
  function themeQuery(host) {
    var q = '';
    try {
      var rb = host.querySelector('[data-hbt=head] [data-role]');
      if (rb && /Caregiver/.test(rb.textContent)) q += '&cg=1';
    } catch (e) {}
    return q;
  }
  function openDrawer(href, title, trigger, fit) {
    closeDrawer(false);
    var host = frame() || document.body;
    var el = document.createElement('div');
    el.className = 'hbt-dr';
    var sep = href.indexOf('?') >= 0 ? '&' : '?';
    var src = href + sep + 'embed=1' + (fit ? '&fit=1' : '') + themeQuery(host);
    el.innerHTML = '<div class="hbt-dr-p" role="dialog" aria-modal="true" aria-label="' + title.replace(/"/g, '&quot;') + '"><div class="hbt-dr-grab"></div><div class="hbt-dr-h"><h2 class="hbt-dr-t"></h2><button type="button" class="hbt-dr-x" aria-label="Close"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg></button></div><iframe class="hbt-dr-f" title="' + title.replace(/"/g, '&quot;') + '"></iframe></div>';
    el.querySelector('.hbt-dr-t').textContent = title;
    if (fit) el.querySelector('.hbt-dr-p').className += ' hbt-dr-fit';
    el.addEventListener('click', function (e) { if (e.target === el) closeDrawer(); });
    el.querySelector('.hbt-dr-x').addEventListener('click', function () { closeDrawer(); });
    var key = function (e) { if (e.key === 'Escape') { e.stopPropagation(); closeDrawer(); } };
    document.addEventListener('keydown', key, true);
    host.appendChild(el);
    var fr = el.querySelector('iframe');
    cur = { el: el, key: key, trigger: trigger, frame: fr };
    fr.src = src;
    fr.addEventListener('load', function () { try { fr.contentWindow.focus(); } catch (e) {} });
    el.querySelector('.hbt-dr-x').focus();
  }
  document.addEventListener('click', function (e) {
    if (e.defaultPrevented && !e.target.closest) return;
    var a = e.target.closest && e.target.closest('a[data-drawer]');
    if (!a || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey || e.button) return;
    var title = a.getAttribute('data-drawer');
    if (!title) return;
    e.preventDefault();
    var href = a.getAttribute('href');
    /* let the link's own handler record the selection first, then open */
    var fit = a.hasAttribute('data-drawer-fit');
    setTimeout(function () { openDrawer(href, title, a, fit); }, 0);
  }, true);
  window.addEventListener('message', function (e) {
    var d = e.data;
    if (!d || !cur || e.source !== cur.frame.contentWindow) return;
    if (d.hbt === 'size') { cur.frame.style.height = d.h + 'px'; cur.frame.style.flex = 'none'; return; }
    if (d.hbt !== 'close') return;
    closeDrawer();
    if (d.go) { location.href = d.go; return; }
    if (d.reload) { location.reload(); return; }
    if (d.saved) {
      try { sessionStorage.setItem('hbt-toast', d.saved); } catch (x) {}
      location.reload();
    }
  });
  /* confirmation after the list page reloads with fresh data */
  var tmsg = null;
  try { tmsg = sessionStorage.getItem('hbt-toast'); sessionStorage.removeItem('hbt-toast'); } catch (e) {}
  if (tmsg) {
    var tries = 0, iv = setInterval(function () {
      var f = frame();
      if (f || ++tries > 40) {
        clearInterval(iv);
        if (!f) return;
        var t = document.createElement('div');
        t.className = 'hbt-toast'; t.setAttribute('role', 'status'); t.textContent = tmsg;
        f.appendChild(t);
        setTimeout(function () { t.parentNode && t.parentNode.removeChild(t); }, 5000);
      }
    }, 100);
  }
})();
