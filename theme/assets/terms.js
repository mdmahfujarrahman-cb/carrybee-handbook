// Courier terms page: word-start search (skipping the brand word), category chips,
// "Where it lives" links into the app maps and #t.<id> deep links with a flash.
// Ported from the prototype terms page; reads HB.terms from data/terms.js.
(function () {
  'use strict';

  const HB = window.HB || {};
  const DATA = HB.terms;
  const out = document.getElementById('terms-out');
  const input = document.getElementById('terms-q');
  const catBox = document.getElementById('terms-cats');
  const countEl = document.getElementById('terms-count');
  if (!DATA || !out || !input || !catBox) return;

  const root = HB.root || '';
  const MOD_DESC = DATA.modules || {};
  const APPS = (HB.sync?.apps || []).map(a => a.id);
  if (!APPS.length) APPS.push(...Object.keys(MOD_DESC));
  const BRAND = /carrybee/gi;
  const ESC_MAP = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' };
  const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ESC_MAP[c]);
  const reEsc = s => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const akaOf = t => t.aka || [];
  const whereOf = t => t.where || {};
  let cat = 'all';

  // Highlights matches of q, but never inside the word "carrybee".
  function mark(text, q) {
    const safe = esc(text);
    if (!q) return safe;
    return safe.replace(new RegExp(reEsc(esc(q)), 'ig'), (m, at, all) => {
      const word = all.slice(0, at).match(/[a-z]*$/i)[0] + m + all.slice(at + m.length).match(/^[a-z]*/i)[0];
      return word.toLowerCase() === 'carrybee' ? m : '<mark>' + m + '</mark>';
    });
  }

  // -1 = no match, 0 = every word matches the name or a code name, 1 = matches the definition only.
  function rank(t, q) {
    if (!q) return 0;
    const words = q.toLowerCase().split(/\s+/);
    const name = [t.t, ...akaOf(t)].join(' ').toLowerCase();
    const all = name + ' ' + t.def.replace(BRAND, '').toLowerCase();
    const has = (hay, w) => new RegExp('(^|[^a-z0-9])' + reEsc(w)).test(hay);
    if (!words.every(w => has(all, w))) return -1;
    return words.every(w => has(name, w)) ? 0 : 1;
  }
  const matches = (t, q) => rank(t, q) >= 0;

  function whereLinks(t) {
    const where = whereOf(t);
    const links = APPS.filter(a => where[a]).flatMap(a => where[a].map(m => {
      const d = (MOD_DESC[a] || {})[m];
      const tip = d ? ` title="${esc(d[0] + ': ' + d[1])}"` : '';
      return `<a class="where ${esc(a)}" href="${esc(root)}apps/${esc(a)}.html#mod.${esc(m)}"${tip}><b>${esc(a)}</b><span>${esc(m)}</span></a>`;
    }));
    return links.length ? links.join('') : '<span class="none">Not linked to a module yet</span>';
  }

  function openFromHash() {
    const m = /^#t\.([a-z0-9-]+)$/.exec(location.hash);
    if (!m) return;
    const el = document.getElementById('t.' + m[1]);
    if (!el) return;
    el.scrollIntoView({ block: 'center' });
    el.classList.add('flash');
  }

  function renderCats(q) {
    const counts = {};
    DATA.terms.filter(t => matches(t, q)).forEach(t => { counts[t.cat] = (counts[t.cat] || 0) + 1; });
    const total = Object.values(counts).reduce((a, b) => a + b, 0);
    const btn = (id, name, n) => `<button type="button" data-cat="${esc(id)}" aria-pressed="${cat === id}">${esc(name)}<span>${n}</span></button>`;
    catBox.innerHTML = btn('all', 'All', total) + DATA.categories.map(c => btn(c.id, c.name, counts[c.id] || 0)).join('');
  }

  function termRow(t, q) {
    return `<div class="term-row" id="t.${esc(t.id)}"><dt>${mark(t.t, q)}${t.inferred ? '<span class="flag">inferred</span>' : ''}</dt><dd>
      <p>${mark(t.def, q)}</p>
      <div class="meta"><em>In code</em>${akaOf(t).map(a => `<span class="code">${mark(a, q)}</span>`).join('')}</div>
      <div class="meta"><em>Where it lives</em>${whereLinks(t)}</div>
    </dd></div>`;
  }

  function render() {
    const q = input.value.trim();
    renderCats(q);
    let shown = 0;
    const html = DATA.categories.filter(c => cat === 'all' || c.id === cat).map(c => {
      const terms = DATA.terms.filter(t => t.cat === c.id && matches(t, q)).sort((a, b) => rank(a, q) - rank(b, q));
      shown += terms.length;
      if (!terms.length) return '';
      return `<section class="term-group" aria-label="${esc(c.name)}"><h2>${esc(c.name)}<span>${terms.length}</span></h2>` +
        `<dl class="term-list">${terms.map(t => termRow(t, q)).join('')}</dl></section>`;
    }).join('');
    out.innerHTML = html || `<p class="terms-empty">No term matches “${esc(q)}”. Try a shorter word or another category.</p>`;
    if (countEl) countEl.textContent = `${shown} of ${DATA.terms.length} terms`;
  }

  input.addEventListener('input', render);
  catBox.addEventListener('click', e => {
    const b = e.target.closest('button[data-cat]');
    if (!b) return;
    cat = b.dataset.cat;
    render();
  });
  render();
  openFromHash();
  addEventListener('hashchange', () => {
    input.value = '';
    cat = 'all';
    render();
    openFromHash();
  });
})();
