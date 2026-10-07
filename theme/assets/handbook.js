// Carrybee Handbook: theme toggle, drawer, ⌘K search, term popovers, "On this page" and the Sync pill.
// All data comes from data/*.js files loaded before this script, so the site works from file:// too.
(function () {
  'use strict';

  const HB = window.HB || {};
  const root = HB.root || '';
  const $ = (sel, ctx) => (ctx || document).querySelector(sel);
  const $$ = (sel, ctx) => Array.from((ctx || document).querySelectorAll(sel));
  const ESC_MAP = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' };
  const escapeHTML = s => String(s ?? '').replace(/[&<>"']/g, c => ESC_MAP[c]);
  const escapeRE = s => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const isMac = /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent);

  /* ---------- Theme ---------- */
  const THEME_KEY = 'hb-theme';
  const darkQuery = window.matchMedia('(prefers-color-scheme: dark)');
  const currentTheme = () => document.documentElement.dataset.theme || (darkQuery.matches ? 'dark' : 'light');

  function labelThemeButton(btn) {
    btn.setAttribute('aria-label', currentTheme() === 'dark' ? 'Switch to light theme' : 'Switch to dark theme');
  }

  function initTheme() {
    const btn = $('#theme-toggle');
    if (!btn) return;
    labelThemeButton(btn);
    darkQuery.addEventListener('change', () => labelThemeButton(btn));
    btn.addEventListener('click', () => {
      const next = currentTheme() === 'dark' ? 'light' : 'dark';
      document.documentElement.dataset.theme = next;
      try {
        localStorage.setItem(THEME_KEY, next);
      } catch (err) {
        console.warn('Theme choice not saved; it lasts for this page only.', err);
      }
      labelThemeButton(btn);
    });
  }

  /* ---------- Drawer (≤ 900px) ---------- */
  function initDrawer() {
    const btn = $('#menu-toggle');
    const scrim = $('.scrim');
    if (!btn || !scrim) return;
    const setOpen = open => {
      document.body.classList.toggle('nav-open', open);
      btn.setAttribute('aria-expanded', String(open));
      btn.setAttribute('aria-label', open ? 'Close navigation' : 'Open navigation');
      scrim.hidden = !open;
      if (open) ($('.sidebar a[aria-current="page"]') || $('.sidebar a'))?.focus({ preventScroll: true });
    };
    btn.addEventListener('click', () => setOpen(!document.body.classList.contains('nav-open')));
    scrim.addEventListener('click', () => setOpen(false));
    document.addEventListener('keydown', e => {
      if (e.key === 'Escape' && document.body.classList.contains('nav-open')) {
        setOpen(false);
        btn.focus();
      }
    });
  }

  /* ---------- Search ---------- */
  const GROUPS = [
    ['page', 'Handbook pages'], ['term', 'Courier terms'], ['module', 'Modules'],
    ['route', 'App pages'], ['status', 'Statuses'], ['api', 'API domains'],
  ];
  const PER_GROUP = 6;
  const IGNORED_WORDS = new Set(['carrybee']);
  const index = (HB.search || []).map(e => ({
    ...e,
    t: e.title.toLowerCase(),
    h: [e.title, e.sub, e.text].join(' ').toLowerCase(),
  }));

  const queryWords = q => q.toLowerCase().split(/\s+/).filter(w => w && !IGNORED_WORDS.has(w));

  // A query word must match the start of a word ("hub" finds "hubs", "run" does not find "trunk").
  const wordStart = w => new RegExp('(^|[^a-z0-9])' + escapeRE(w));

  function score(entry, words, res) {
    let s = entry.t === words.join(' ') ? 20 : 0;
    for (const re of res) {
      const m = re.exec(entry.h);
      if (!m) return -1;
      const inTitle = re.exec(entry.t);
      if (inTitle && inTitle.index === 0) s += 6;
      else if (inTitle) s += 3;
      else s += 1;
    }
    return s;
  }

  function highlight(text, words) {
    if (!words.length || !text) return escapeHTML(text);
    const re = new RegExp('(^|[^a-z0-9])(' + words.map(escapeRE).join('|') + ')', 'gi');
    let out = '';
    let last = 0;
    for (const m of text.matchAll(re)) {
      const start = m.index + m[1].length;
      out += escapeHTML(text.slice(last, start)) + '<mark>' + escapeHTML(m[2]) + '</mark>';
      last = start + m[2].length;
    }
    return out + escapeHTML(text.slice(last));
  }

  function findResults(q) {
    const words = queryWords(q);
    if (!words.length) return { words, hits: index.filter(e => e.type === 'page').map(e => ({ e, s: 0 })) };
    const res = words.map(wordStart);
    const hits = [];
    for (const e of index) {
      const s = score(e, words, res);
      if (s >= 0) hits.push({ e, s });
    }
    hits.sort((a, b) => b.s - a.s || a.e.title.localeCompare(b.e.title));
    return { words, hits };
  }

  function renderGroup(label, items, words, start) {
    let html = '<div class="sr-group" role="presentation">' + escapeHTML(label) + '</div>';
    items.forEach(({ e }, i) => {
      html += '<a class="sr-item" role="option" aria-selected="false" id="sr-' + (start + i) + '" href="' +
        escapeHTML(root + e.href) + '"><b>' + highlight(e.title, words) + '</b>' +
        (e.sub ? '<small>' + highlight(e.sub, words) + '</small>' : '') + '</a>';
    });
    return html;
  }

  function initSearch() {
    const dialog = $('#search');
    const input = $('#search-input');
    const list = $('#search-results');
    const count = $('.search-count');
    const opener = $('#search-open');
    if (!dialog || !input || !list) return;
    $$('[data-kbd]').forEach(k => { k.textContent = isMac ? '⌘K' : 'Ctrl K'; });
    let active = -1;

    const options = () => $$('.sr-item', list);
    const setActive = i => {
      const opts = options();
      if (!opts.length) { active = -1; input.removeAttribute('aria-activedescendant'); return; }
      active = (i + opts.length) % opts.length;
      opts.forEach((o, n) => o.setAttribute('aria-selected', String(n === active)));
      input.setAttribute('aria-activedescendant', opts[active].id);
      opts[active].scrollIntoView({ block: 'nearest' });
    };

    const render = () => {
      const { words, hits } = findResults(input.value);
      let html = '';
      let n = 0;
      for (const [type, label] of GROUPS) {
        const items = hits.filter(h => h.e.type === type).slice(0, PER_GROUP);
        if (!items.length) continue;
        html += renderGroup(label, items, words, n);
        n += items.length;
      }
      list.innerHTML = html || '<p class="sr-empty">Nothing matches “' + escapeHTML(input.value.trim()) +
        '”. Try a shorter word, a module name or a status number.</p>';
      if (count) count.textContent = words.length ? hits.length + (hits.length === 1 ? ' result' : ' results') : '';
      setActive(0);
    };

    const open = () => {
      if (dialog.open) return;
      dialog.showModal();
      render();
      input.select();
    };
    opener?.addEventListener('click', open);
    $('[data-close-search]', dialog)?.addEventListener('click', () => dialog.close());
    dialog.addEventListener('click', e => { if (e.target === dialog) dialog.close(); });
    dialog.addEventListener('close', () => opener?.focus());
    input.addEventListener('input', render);
    list.addEventListener('mousemove', e => {
      const item = e.target.closest('.sr-item');
      if (item) setActive(options().indexOf(item));
    });
    list.addEventListener('click', e => { if (e.target.closest('.sr-item')) dialog.close(); });
    input.addEventListener('keydown', e => {
      if (e.key === 'ArrowDown') { e.preventDefault(); setActive(active + 1); }
      else if (e.key === 'ArrowUp') { e.preventDefault(); setActive(active - 1); }
      else if (e.key === 'Enter' && active >= 0) {
        e.preventDefault();
        const target = options()[active];
        dialog.close();
        location.href = target.href;
      }
    });
    document.addEventListener('keydown', e => {
      const typing = /^(INPUT|TEXTAREA|SELECT)$/.test(document.activeElement?.tagName || '');
      if ((e.key === 'k' || e.key === 'K') && (e.metaKey || e.ctrlKey)) { e.preventDefault(); open(); }
      else if (e.key === '/' && !typing && !dialog.open) { e.preventDefault(); open(); }
    });
  }

  /* ---------- Term popovers ---------- */
  function termMap() {
    const map = new Map();
    for (const t of (HB.terms && HB.terms.terms) || []) map.set(t.id, t);
    return map;
  }

  function whereLine(term) {
    const parts = [];
    for (const [app, mods] of Object.entries(term.where || {})) {
      for (const m of mods) parts.push(app + '/' + m);
    }
    return parts.length ? 'Lives in ' + parts.slice(0, 4).join(', ') : '';
  }

  function placeTip(tip, target) {
    const r = target.getBoundingClientRect();
    const w = tip.offsetWidth;
    const h = tip.offsetHeight;
    tip.style.left = Math.min(Math.max(8, r.left + r.width / 2 - w / 2), innerWidth - w - 8) + 'px';
    tip.style.top = (r.top - h - 8 < 8 ? r.bottom + 8 : r.top - h - 8) + 'px';
  }

  function initTerms() {
    const tip = $('#tip');
    const terms = termMap();
    if (!tip || !terms.size) return;
    const termsPage = id => root + 'terms.html#t.' + encodeURIComponent(id);
    $$('.term[data-term]').forEach(el => {
      el.tabIndex = 0;
      el.setAttribute('role', 'link');
    });
    const show = el => {
      const t = terms.get(el.dataset.term);
      if (!t) return;
      tip.replaceChildren();
      const name = document.createElement('b');
      name.textContent = t.t;
      tip.append(name, t.def);
      const where = whereLine(t);
      if (where) {
        const w = document.createElement('span');
        w.className = 'tip-where';
        w.textContent = where;
        tip.append(w);
      }
      tip.hidden = false;
      el.setAttribute('aria-describedby', 'tip');
      placeTip(tip, el);
    };
    const hide = () => { tip.hidden = true; };
    const termAt = e => e.target.closest?.('.term[data-term]');
    document.addEventListener('mouseover', e => { const el = termAt(e); if (el) show(el); });
    document.addEventListener('mouseout', e => { if (termAt(e)) hide(); });
    document.addEventListener('focusin', e => { const el = termAt(e); if (el) show(el); });
    document.addEventListener('focusout', hide);
    document.addEventListener('scroll', hide, true);
    document.addEventListener('click', e => { const el = termAt(e); if (el) location.href = termsPage(el.dataset.term); });
    document.addEventListener('keydown', e => {
      const el = termAt(e);
      if (e.key === 'Escape') hide();
      else if (el && e.key === 'Enter') location.href = termsPage(el.dataset.term);
    });
  }

  /* ---------- On this page ---------- */
  const slug = s => s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '') || 'section';

  function initToc() {
    const toc = $('#toc');
    const heads = $$('#content .prose h2');
    if (!toc || heads.length < 2) return;
    const used = new Set();
    let html = '<h2>On this page</h2><ol>';
    for (const h of heads) {
      if (!h.id) {
        let id = slug(h.textContent);
        while (used.has(id) || document.getElementById(id)) id += '-x';
        h.id = id;
      }
      used.add(h.id);
      html += '<li><a href="#' + escapeHTML(h.id) + '">' + escapeHTML(h.textContent) + '</a></li>';
    }
    toc.innerHTML = html + '</ol>';
    toc.hidden = false;
    const links = new Map($$('a', toc).map(a => [a.getAttribute('href').slice(1), a]));
    const visible = new Set();
    const mark = () => {
      const current = heads.find(h => visible.has(h.id));
      if (!current) return;
      links.forEach(a => a.removeAttribute('aria-current'));
      links.get(current.id)?.setAttribute('aria-current', 'true');
    };
    const io = new IntersectionObserver(entries => {
      for (const en of entries) {
        if (en.isIntersecting) visible.add(en.target.id);
        else visible.delete(en.target.id);
      }
      mark();
    }, { rootMargin: '-64px 0px -40% 0px' });
    heads.forEach(h => io.observe(h));
  }

  /* ---------- Sync pill (deployed mode; local mode is added by the Sync server task) ---------- */
  function relTime(iso) {
    const secs = Math.max(0, (Date.now() - new Date(iso).getTime()) / 1000);
    if (!Number.isFinite(secs)) return 'at an unknown time';
    if (secs < 60) return 'just now';
    if (secs < 3600) return Math.round(secs / 60) + ' min ago';
    if (secs < 86400) return Math.round(secs / 3600) + ' h ago';
    return Math.round(secs / 86400) + ' d ago';
  }

  const shortCommit = c => (c ? c.slice(0, 8) : 'no commit');

  function syncPanelHTML(sync) {
    const built = new Date(sync.builtAt);
    let html = '<h2>Site build</h2><p class="built">Built ' + escapeHTML(built.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })) +
      ' (' + relTime(sync.builtAt) + ')</p><ul>';
    for (const a of sync.apps || []) {
      const detail = a.error
        ? 'Not scanned: ' + a.error
        : a.modules + ' modules, ' + a.pages + ' pages, scanned ' + relTime(a.scannedAt);
      html += '<li' + (a.error ? ' class="err"' : '') + '><b>' + escapeHTML(a.label) + '</b><code>' +
        escapeHTML(shortCommit(a.commit)) + '</code><small>' + escapeHTML(detail) + '</small></li>';
    }
    return html + '</ul><p class="foot">This site rebuilds automatically on every push.</p>';
  }

  function initSync() {
    const pill = $('#sync-pill');
    const panel = $('#sync-panel');
    const sync = HB.sync;
    if (!pill || !panel || !sync) return;
    const first = (sync.apps || [])[0];
    const text = $('.sync-text', pill);
    text.innerHTML = 'Synced ' + escapeHTML(relTime(sync.builtAt)) +
      (first ? ' · ' + escapeHTML(first.label) + ' <span class="mono">' + escapeHTML(shortCommit(first.commit)) + '</span>' : '');
    panel.innerHTML = syncPanelHTML(sync);
    const setOpen = open => {
      panel.hidden = !open;
      pill.setAttribute('aria-expanded', String(open));
    };
    pill.addEventListener('click', () => {
      if (pill.dataset.mode !== 'local') setOpen(panel.hidden);
    });
    document.addEventListener('click', e => {
      if (!panel.hidden && !e.target.closest('.sync')) setOpen(false);
    });
    document.addEventListener('keydown', e => {
      if (e.key === 'Escape' && !panel.hidden) { setOpen(false); pill.focus(); }
    });
  }

  /* ---------- Local Sync (only when served by `handbook serve`) ----------
     The only fetch() calls in the theme: they hit the local server's /api/status and /api/sync. */
  const plural = (n, word) => n + ' ' + word + (n === 1 ? '' : 's');

  function summariseDiff(diff, missing) {
    const parts = (diff || []).map(d => {
      const bits = [];
      const add = (n, sign, word) => { if (n) bits.push(sign + plural(n, word)); };
      add((d.addedModules || []).length, '+', 'module');
      add((d.removedModules || []).length, '-', 'module');
      add((d.addedPages || []).length, '+', 'page');
      add((d.removedPages || []).length, '-', 'page');
      add((d.addedApis || []).length, '+', 'API');
      add((d.removedApis || []).length, '-', 'API');
      return d.app + ': ' + bits.join(', ');
    });
    let msg = parts.length ? parts.join('; ') : 'Already up to date';
    if ((missing || []).length) msg += ' \u00b7 ' + missing.length + ' need descriptions';
    return msg;
  }

  async function runSync(pill, text) {
    const label = text.textContent;
    pill.disabled = true;
    pill.dataset.state = 'busy';
    text.textContent = 'Syncing\u2026';
    let reload = false;
    try {
      const res = await fetch(root + 'api/sync', { method: 'POST' });
      const body = await res.json().catch(() => ({}));
      if (res.status === 409) {
        toast('A sync is already running');
      } else if (res.ok && body.ok) {
        toast(summariseDiff(body.diff, body.missing), 1500);
        reload = true;
        setTimeout(() => location.reload(), 1500);
      } else {
        pill.dataset.state = 'error';
        toast('Sync failed: ' + (body.error || res.statusText || 'unknown error'), 8000);
      }
    } catch (err) {
      pill.dataset.state = 'error';
      toast('Sync failed: ' + err.message, 8000);
    }
    if (!reload) {
      pill.disabled = false;
      text.textContent = label;
      if (pill.dataset.state === 'busy') delete pill.dataset.state;
    }
  }

  async function initLocalSync() {
    const pill = $('#sync-pill');
    if (!pill || !location.protocol.startsWith('http')) return;
    try {
      const res = await fetch(root + 'api/status');
      const status = res.ok ? await res.json() : null;
      if (!status || status.mode !== 'local') return;
    } catch (_) {
      return; // deployed or file:// without an API: keep the read-only pill
    }
    const text = $('.sync-text', pill);
    const panel = $('#sync-panel');
    if (panel) panel.hidden = true;
    pill.dataset.mode = 'local';
    pill.setAttribute('aria-expanded', 'false');
    pill.removeAttribute('aria-controls');
    pill.title = 'Rescan the code and rebuild this handbook';
    text.textContent = 'Sync';
    pill.addEventListener('click', () => { if (!pill.disabled) runSync(pill, text); });
  }

  /* ---------- Toast (used by local Sync) ---------- */
  function toast(msg, ms) {
    const el = $('#toast');
    if (!el) return;
    el.textContent = msg;
    el.hidden = false;
    clearTimeout(toast.timer);
    toast.timer = setTimeout(() => { el.hidden = true; }, ms || 4000);
  }

  HB.ui = { toast, relTime };
  initTheme();
  initDrawer();
  initSearch();
  initTerms();
  initToc();
  initSync();
  initLocalSync();
})();
