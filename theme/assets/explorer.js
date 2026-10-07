// App map explorer: A-to-Z lifecycle, page/module flow and the whole module network.
// Ported from the prototype module maps; reads HB.app (data/<app>.js) and reuses the layout's #tip.
(function () {
  'use strict';

  const HB = window.HB || {};
  const APP = HB.app;
  if (!APP || APP.error || !document.getElementById('main')) return;

  const ESC_MAP = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' };
  const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ESC_MAP[c]);

  const DATA = { mods: APP.mods || {}, routes: APP.routes || {}, cycles: APP.cycles || [] };
  const DESC = APP.desc || {};
  const STORY = APP.story && APP.story.length ? APP.story : null;
  const GROUPS = APP.groups || {};
  const TERMLINKS = APP.termLinks || {};
  const STORE_KEY = 'hb-map-' + APP.id;

  const groupInfo = g => GROUPS[g] ? `<span class="info" data-group="${esc(g)}" aria-label="About ${esc(g)}">i</span>` : '';
  const tipEl = () => document.getElementById('tip');
  const info = m => DESC[m] ? `<span class="info" data-mod="${esc(m)}" aria-label="About ${esc(m)}">i</span>` : '';
  function showTip(target) {
    const d = target.dataset.group ? GROUPS[target.dataset.group] : DESC[target.dataset.mod];
    const tip = tipEl();
    if (!d || !tip) return;
    tip.innerHTML = `<b>${esc(d[0])}</b>${esc(d[1])}`;
    tip.hidden = false;
    const r = target.getBoundingClientRect();
    const w = tip.offsetWidth, hgt = tip.offsetHeight;
    const left = Math.min(Math.max(8, r.left + r.width / 2 - w / 2), window.innerWidth - w - 8);
    const top = r.top - hgt - 8 < 8 ? r.bottom + 8 : r.top - hgt - 8;
    tip.style.left = left + 'px';
    tip.style.top = top + 'px';
  }
  const hideTip = () => { const tip = tipEl(); if (tip) tip.hidden = true; };
  document.addEventListener('mouseover', e => { const t = e.target.closest?.('.info'); if (t) showTip(t); });
  document.addEventListener('mouseout', e => { if (e.target.closest?.('.info')) hideTip(); });
  document.addEventListener('scroll', hideTip, true);

  const MODS = DATA.mods, ROUTES = DATA.routes;
  const cycleOf = {};
  DATA.cycles.forEach((c, i) => c.forEach(m => { cycleOf[m] = i; }));
  const sameCycle = (a, b) => cycleOf[a] !== undefined && cycleOf[a] === cycleOf[b];
  const byWeight = obj => Object.entries(obj || {}).sort((a, b) => b[1] - a[1]).map(e => e[0]);
  const routeLabel = r => {
    const m = /^\(([^)]+)\)\/?(.*)$/.exec(r);
    if (!m) return { name: r === '/' ? '/' : '/' + r, group: '' };
    return { name: m[2] ? '/' + m[2] : '(' + m[1] + ')', group: m[1] };
  };
  const MODULE_KIND = 'mod';
  const ROUTE_KIND = 'route';

  // The default page is the busiest page of the first module in the first story step, else the first route.
  function defaultRoute() {
    const firstMod = STORY?.[0]?.steps?.[0]?.mods?.find(m => MODS[m]);
    const page = firstMod ? byWeight(MODS[firstMod].usedByRoutes).find(r => ROUTES[r]) : undefined;
    return page ?? Object.keys(ROUTES)[0];
  }

  let state = { tab: ROUTE_KIND, sel: null, view: 'flow' };
  try { const s = JSON.parse(localStorage.getItem(STORE_KEY) || 'null'); if (s) state = { ...state, ...s, view: 'flow' }; } catch (e) { console.warn('stored state unreadable', e); }
  const save = () => { try { localStorage.setItem(STORE_KEY, JSON.stringify({ tab: state.tab, sel: state.sel })); } catch (e) { console.warn('could not store state', e); } };

  const el = (tag, cls, html) => { const e = document.createElement(tag); if (cls) e.className = cls; if (html !== undefined) e.innerHTML = html; return e; };

  function renderList() {
    const q = document.getElementById('q').value.trim().toLowerCase();
    const list = document.getElementById('list');
    list.innerHTML = '';
    if (state.tab === ROUTE_KIND) {
      const groups = {};
      Object.keys(ROUTES).filter(r => r.toLowerCase().includes(q)).forEach(r => {
        const g = routeLabel(r).group || 'top level';
        (groups[g] = groups[g] || []).push(r);
      });
      Object.keys(groups).sort().forEach(g => {
        list.append(el('div', 'grp', esc(g) + groupInfo(g)));
        groups[g].sort().forEach(r => list.append(item(ROUTE_KIND, r, esc(routeLabel(r).name), Object.keys(ROUTES[r].mods || {}).length + ' mod')));
      });
    } else {
      Object.keys(MODS).filter(m => m.includes(q)).sort().forEach(m => {
        const fanIn = Object.keys(MODS[m].usedByMods || {}).length;
        const name = (cycleOf[m] === undefined ? esc(m) : esc(m) + ' <span class="cyc-dot">●</span>') + info(m);
        list.append(item(MODULE_KIND, m, name, fanIn ? '←' + fanIn : ''));
      });
    }
    revealSelected(list);
  }
  // Scrolls only the picker list (not the page) so the selected row is visible.
  function revealSelected(list) {
    const on = list.querySelector('button.on');
    if (!on) return;
    const top = on.offsetTop - list.offsetTop;
    if (top < list.scrollTop || top + on.offsetHeight > list.scrollTop + list.clientHeight) {
      list.scrollTop = top - list.clientHeight / 2;
    }
  }
  function item(kind, id, name, meta) {
    const b = el('button', state.sel?.kind === kind && state.sel.id === id ? 'on' : '', `<span>${name}</span><small>${meta}</small>`);
    b.type = 'button';
    b.addEventListener('click', () => select(kind, id));
    return b;
  }

  function select(kind, id) {
    state.sel = { kind, id };
    if (state.tab !== kind) setTab(kind);
    setView('flow');
    save();
    renderList();
    renderFlow();
  }

  function nodeBtn(kind, id, opts = {}) {
    const label = kind === ROUTE_KIND ? routeLabel(id).name : id;
    const cls = ['node', kind, opts.focus ? 'focus' : '', opts.dim ? 'dim' : '', kind === MODULE_KIND && cycleOf[id] !== undefined ? 'cyc' : ''].join(' ');
    const b = el('button', cls, esc(label) + (kind === MODULE_KIND ? info(id) : '') + (opts.note ? `<small>${esc(opts.note)}</small>` : ''));
    b.type = 'button';
    b.dataset.key = kind + ':' + id;
    if (kind === 'api') b.disabled = true; else b.addEventListener('click', () => select(kind, id));
    return b;
  }

  function moduleColumns(m) {
    const mod = MODS[m];
    const deps = byWeight(mod.deps);
    const apis = {};
    Object.keys(mod.apis || {}).forEach(a => { apis[a] = { via: null }; });
    deps.forEach(d => Object.keys(MODS[d]?.apis || {}).forEach(a => { if (!apis[a]) apis[a] = { via: d }; }));
    const edges = [];
    const pages = byWeight(mod.usedByRoutes), ups = byWeight(mod.usedByMods);
    pages.forEach(r => edges.push(['route:' + r, 'mod:' + m]));
    ups.forEach(u => edges.push(['mod:' + u, 'mod:' + m, sameCycle(u, m)]));
    deps.forEach(d => edges.push(['mod:' + m, 'mod:' + d, sameCycle(d, m)]));
    Object.entries(apis).forEach(([a, v]) => edges.push([v.via ? 'mod:' + v.via : 'mod:' + m, 'api:' + a]));
    return {
      title: 'Module · ' + m,
      hint: `${mod.files} files in src/modules/${m}. Left: who reaches it. Right: what it pulls in and which backend APIs end up being called.`,
      cols: [
        ['Pages that use it', pages.map(r => nodeBtn(ROUTE_KIND, r, { note: routeLabel(r).group }))],
        ['Imported by modules', ups.map(u => nodeBtn(MODULE_KIND, u))],
        ['Module', [nodeBtn(MODULE_KIND, m, { focus: true, note: mod.files + ' files' })]],
        ['Imports modules', deps.map(d => nodeBtn(MODULE_KIND, d))],
        ['Backend API', Object.entries(apis).map(([a, v]) => nodeBtn('api', a, { dim: !!v.via, note: v.via ? 'via ' + v.via : 'direct' }))],
      ],
      edges, shared: mod.shared,
    };
  }

  function routeColumns(r) {
    const direct = byWeight(ROUTES[r].mods);
    const directSet = new Set(direct);
    const indirect = {}, apis = {}, edges = [];
    direct.forEach(m => {
      edges.push(['route:' + r, 'mod:' + m]);
      byWeight(MODS[m]?.deps).forEach(d => {
        if (directSet.has(d)) return;
        edges.push(['mod:' + m, 'mod2:' + d]);
        indirect[d] = indirect[d] || m;
      });
    });
    const addApis = (m, prefix) => Object.keys(MODS[m]?.apis || {}).forEach(a => {
      if (!apis[a]) { apis[a] = prefix === 'mod' ? null : m; edges.push([prefix + ':' + m, 'api:' + a]); }
    });
    direct.forEach(m => addApis(m, 'mod'));
    Object.keys(indirect).forEach(m => addApis(m, 'mod2'));
    const ind = Object.keys(indirect).sort().map(d => { const b = nodeBtn(MODULE_KIND, d, { note: 'via ' + indirect[d] }); b.dataset.key = 'mod2:' + d; return b; });
    const lbl = routeLabel(r);
    return {
      title: 'Page · ' + lbl.name,
      hint: `${lbl.group ? 'Route group (' + lbl.group + ') · ' : ''}${ROUTES[r].files} files under src/app. Follow it left to right to see every module the page loads and the backend APIs it can call.`,
      cols: [
        ['Page', [nodeBtn(ROUTE_KIND, r, { focus: true, note: lbl.group })]],
        ['Modules it uses', direct.map(m => nodeBtn(MODULE_KIND, m))],
        ['Pulled in indirectly', ind],
        ['Backend API', Object.entries(apis).map(([a, via]) => nodeBtn('api', a, { dim: !!via, note: via ? 'via ' + via : 'direct' }))],
      ],
      edges, shared: null,
    };
  }

  function renderFlow() {
    const flow = document.getElementById('flow');
    const notes = document.getElementById('notes');
    flow.innerHTML = ''; notes.innerHTML = '';
    if (!state.sel || !(state.sel.kind === ROUTE_KIND ? ROUTES : MODS)[state.sel.id]) {
      state.sel = { kind: ROUTE_KIND, id: defaultRoute() };
    }
    if (state.sel.id === undefined) return;
    const spec = state.sel.kind === MODULE_KIND ? moduleColumns(state.sel.id) : routeColumns(state.sel.id);
    document.getElementById('f-title').textContent = spec.title;
    document.getElementById('f-hint').textContent = spec.hint;
    const descBox = document.getElementById('f-desc');
    const d = state.sel.kind === MODULE_KIND ? DESC[state.sel.id] : null;
    descBox.hidden = !d;
    if (d) descBox.innerHTML = `<b>${esc(d[0])}.</b> ${esc(d[1])}`;
    const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    svg.classList.add('wires');
    flow.append(svg);
    spec.cols.forEach(([h, nodes]) => {
      const c = el('div', 'col');
      c.append(el('h3', '', `${esc(h)} <span class="col-n">${nodes.length}</span>`));
      if (nodes.length) nodes.forEach(n => c.append(n)); else c.append(el('div', 'empty', 'none'));
      flow.append(c);
    });
    requestAnimationFrame(() => drawWires(flow, svg, spec.edges));
    if (state.sel.kind === MODULE_KIND) { renderModuleNotes(notes, state.sel.id, spec.shared); renderTermLinks(notes, state.sel.id); }
  }

  function drawWires(flow, svg, edges) {
    const box = flow.getBoundingClientRect();
    const pos = {};
    flow.querySelectorAll('.node').forEach(n => { pos[n.dataset.key] = n.getBoundingClientRect(); });
    svg.innerHTML = edges.map(([a, b, hot]) => {
      const p = pos[a], q = pos[b];
      if (!p || !q) return '';
      const x1 = p.right - box.left, y1 = p.top + p.height / 2 - box.top;
      const x2 = q.left - box.left, y2 = q.top + q.height / 2 - box.top;
      if (x2 <= x1) return '';
      const mx = (x1 + x2) / 2;
      return `<path class="${hot ? 'hot' : ''}" d="M${x1},${y1} C${mx},${y1} ${mx},${y2} ${x2},${y2}"/>`;
    }).join('');
  }

  function renderTermLinks(notes, m) {
    const list = TERMLINKS[m];
    if (!list || !list.length) return;
    const box = el('div', '', '<h4>Courier terms used here</h4>');
    const chips = el('div', 'chips');
    list.forEach(([id, name]) => {
      const a = el('a', 'chip term-link');
      a.textContent = name;
      a.href = (HB.root || '') + 'terms.html#t.' + id;
      chips.append(a);
    });
    box.append(chips);
    notes.prepend(box);
  }
  function openModuleFromHash() {
    const m = /^#mod\.([a-z0-9-]+)$/.exec(location.hash);
    if (m && MODS[m[1]]) select(MODULE_KIND, m[1]);
  }
  function renderModuleNotes(notes, m, shared) {
    const sharedBox = el('div', '', '<h4>Shared code it leans on</h4>');
    const chips = el('div', 'chips');
    byWeight(shared).forEach(s => chips.append(el('span', 'chip', `${esc(s)} · ${shared[s]}`)));
    sharedBox.append(chips);
    notes.append(sharedBox);
    if (cycleOf[m] !== undefined) {
      const box = el('div', '', '<h4 class="cyc-head">In a circular import with</h4>');
      const c = el('div', 'chips');
      DATA.cycles[cycleOf[m]].filter(x => x !== m).forEach(x => {
        const b = el('button', 'chip', esc(x)); b.type = 'button'; b.addEventListener('click', () => select(MODULE_KIND, x)); c.append(b);
      });
      box.append(c); notes.append(box);
    }
  }

  function renderCycles() {
    const box = document.getElementById('cycles');
    if (!DATA.cycles.length) { box.append(el('p', 'empty', 'None. Every module can be changed without a loop back to itself.')); return; }
    DATA.cycles.sort((a, b) => b.length - a.length).forEach((c, i) => {
      const row = el('div', 'cycle', `<b>Loop ${i + 1}</b>`);
      c.forEach(m => { const b = el('button', '', esc(m)); b.type = 'button'; b.addEventListener('click', () => select(MODULE_KIND, m)); row.append(b); });
      box.append(row);
    });
    DATA.cycles.forEach((c, i) => c.forEach(m => { cycleOf[m] = i; }));
  }

  function setPhaseOpen(sec, open) {
    sec.classList.toggle('open', open);
    sec.querySelector('.phase-toggle').setAttribute('aria-expanded', String(open));
    sec.querySelector('.steps').hidden = !open;
  }
  function setAllPhases(open) {
    document.querySelectorAll('#story .phase').forEach(sec => setPhaseOpen(sec, open));
  }
  function phaseHead(phase, letter, sec) {
    const head = el('button', 'phase-toggle');
    head.type = 'button';
    head.setAttribute('aria-expanded', 'false');
    head.setAttribute('aria-controls', 'steps-' + letter);
    const summary = phase.steps.map(st => `<span class="sum-chip">${esc(st.title)}</span>`).join('');
    head.innerHTML = `<span class="phase-head"><span class="phase-titleline"><h3>${esc(phase.title)}</h3>`
      + `<span class="count">${phase.steps.length} step${phase.steps.length === 1 ? '' : 's'}</span></span>`
      + `<p>${esc(phase.when)}</p></span><span class="summary">${summary}</span><span class="chev" aria-hidden="true">▸</span>`;
    head.addEventListener('click', () => setPhaseOpen(sec, !sec.classList.contains('open')));
    return head;
  }
  function renderStory() {
    if (!STORY) return;
    const rail = document.getElementById('story-rail');
    const box = document.getElementById('story');
    document.getElementById('story-sub').textContent = APP.storySub || 'step by step';
    rail.innerHTML = ''; box.innerHTML = '';
    STORY.forEach((phase, i) => {
      const letter = String.fromCodePoint(65 + i);
      const link = el('a', '', `<span>${letter}</span>${esc(phase.title)}`);
      link.href = '#phase-' + letter;
      link.addEventListener('click', e => {
        e.preventDefault();
        const target = document.getElementById('phase-' + letter);
        setPhaseOpen(target, true);
        target.scrollIntoView({ behavior: matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth', block: 'start' });
      });
      rail.append(link);
      const sec = el('section', 'phase');
      sec.id = 'phase-' + letter;
      sec.append(el('div', 'phase-num', letter));
      const right = el('div', '');
      right.append(phaseHead(phase, letter, sec));
      const steps = el('div', 'steps');
      steps.id = 'steps-' + letter;
      steps.hidden = true;
      phase.steps.forEach(st => steps.append(storyStep(st)));
      right.append(steps);
      sec.append(right);
      box.append(sec);
    });
  }
  function stepPages(st) {
    const stem = s => s.replace(/^\(.*?\)\/?/, '').split(/[-/]/)[0].replace(/s$/, '');
    const relevance = r => st.mods.some(m => stem(r) && m.startsWith(stem(r))) ? 0 : 1;
    return [...new Set(st.mods.flatMap(m => byWeight(MODS[m]?.usedByRoutes)))]
      .map((r, i) => [r, relevance(r), i]).sort((a, b) => a[1] - b[1] || a[2] - b[2]).map(x => x[0]).slice(0, 4);
  }
  function storyStep(st) {
    const card = el('article', 'stepcard');
    card.append(el('span', 'actor', esc(st.actor)));
    card.append(el('h4', '', esc(st.title)));
    card.append(el('p', '', esc(st.text)));
    if (st.status) card.append(el('div', 'stat-line', 'status → ' + esc(st.status)));
    const modRow = el('div', 'row', '<em>Modules</em>');
    st.mods.forEach(m => {
      const b = el('button', 'chip-mod', esc(m) + info(m));
      b.type = 'button';
      b.addEventListener('click', () => select(MODULE_KIND, m));
      modRow.append(b);
    });
    card.append(modRow);
    const pages = stepPages(st);
    if (pages.length) {
      const pageRow = el('div', 'row', '<em>Pages</em>');
      pages.forEach(r => {
        const b = el('button', 'chip-page', esc(routeLabel(r).name));
        b.type = 'button';
        b.addEventListener('click', () => select(ROUTE_KIND, r));
        pageRow.append(b);
      });
      card.append(pageRow);
    }
    return card;
  }

  let netBuilt = false;
  function buildNet() {
    if (netBuilt) return;
    if (typeof d3 === 'undefined') {
      document.getElementById('net-offline').hidden = false;
      return;
    }
    netBuilt = true;
    const svg = d3.select('#net');
    const { width, height } = svg.node().getBoundingClientRect();
    const nodes = Object.keys(MODS).map(id => ({ id, fan: Object.keys(MODS[id].usedByMods || {}).length }));
    const links = Object.entries(MODS).flatMap(([a, m]) => Object.keys(m.deps || {}).filter(b => MODS[b]).map(b => ({ source: a, target: b, hot: sameCycle(a, b) })));
    const r = d => 4 + Math.sqrt(d.fan) * 3;
    const g = svg.append('g');
    svg.call(d3.zoom().scaleExtent([0.3, 4]).on('zoom', e => g.attr('transform', e.transform)));
    const link = g.append('g').selectAll('line').data(links).join('line').attr('class', d => d.hot ? 'hot' : '');
    const node = g.append('g').selectAll('g').data(nodes).join('g').style('cursor', 'pointer')
      .on('click', (e, d) => select(MODULE_KIND, d.id));
    node.append('circle').attr('r', r)
      .attr('class', d => cycleOf[d.id] === undefined ? 'n-mod' : 'n-mod n-cyc');
    node.append('text').attr('x', d => r(d) + 3).attr('y', 3).text(d => d.id);
    node.append('title').text(d => `${d.id}: ${DESC[d.id] ? DESC[d.id][0] + '. ' + DESC[d.id][1] + ' ' : ''}Imported by ${d.fan} modules.`);
    const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches;
    const sim = d3.forceSimulation(nodes)
      .force('link', d3.forceLink(links).id(d => d.id).distance(70).strength(0.25))
      .force('charge', d3.forceManyBody().strength(-160))
      .force('center', d3.forceCenter(width / 2, height / 2))
      .force('collide', d3.forceCollide(d => r(d) + 14));
    const tick = () => {
      link.attr('x1', d => d.source.x).attr('y1', d => d.source.y).attr('x2', d => d.target.x).attr('y2', d => d.target.y);
      node.attr('transform', d => `translate(${d.x},${d.y})`);
    };
    if (reduce) { sim.stop(); for (let i = 0; i < 300; i++) sim.tick(); tick(); } else sim.on('tick', tick);
  }

  function setTab(t) {
    state.tab = t;
    document.getElementById('tab-routes').setAttribute('aria-selected', String(t === ROUTE_KIND));
    document.getElementById('tab-mods').setAttribute('aria-selected', String(t === MODULE_KIND));
    renderList();
  }
  function setView(v) {
    state.view = v;
    document.getElementById('main').classList.toggle('no-side', v !== 'flow');
    document.getElementById('v-flow').setAttribute('aria-pressed', String(v === 'flow'));
    document.getElementById('v-net').setAttribute('aria-pressed', String(v === 'net'));
    document.getElementById('v-story').setAttribute('aria-pressed', String(v === 'story'));
    document.getElementById('story-view').hidden = v !== 'story';
    document.getElementById('flow-view').hidden = v !== 'flow';
    document.getElementById('net-view').hidden = v !== 'net';
    if (v === 'net') buildNet();
  }

  document.getElementById('n-mods').textContent = String(Object.keys(MODS).length);
  document.getElementById('tab-routes').addEventListener('click', () => setTab(ROUTE_KIND));
  document.getElementById('tab-mods').addEventListener('click', () => setTab(MODULE_KIND));
  document.getElementById('v-flow').addEventListener('click', () => { setView('flow'); renderFlow(); });
  document.getElementById('v-net').addEventListener('click', () => setView('net'));
  document.getElementById('v-story').addEventListener('click', () => setView('story'));
  document.getElementById('q').addEventListener('input', renderList);
  let resizeTimer;
  window.addEventListener('resize', () => { clearTimeout(resizeTimer); resizeTimer = setTimeout(() => { if (state.view === 'flow') renderFlow(); }, 150); });

  renderCycles();
  renderFlow();
  setTab(state.sel?.kind || state.tab);
  if (STORY) {
    document.getElementById('v-story').hidden = false;
    renderStory();
    document.getElementById('expand-all').addEventListener('click', () => setAllPhases(true));
    document.getElementById('collapse-all').addEventListener('click', () => setAllPhases(false));
    const first = document.querySelector('#story .phase');
    if (first) setPhaseOpen(first, true);
    setView('story');
  }
  openModuleFromHash();
  addEventListener('hashchange', openModuleFromHash);
})();
