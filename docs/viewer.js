(function () {
  var PW = 1120, PH = 1584, PX = 96, PT = 84, PBT = 74;
  var CW = PW - 2 * PX;
  var CH = PH - PT - PBT - 4;
  var PRE_PAD = 34;
  var PRE_END_GAP = 8;

  var matter = document.getElementById('matter');
  if (!matter) return;

  var bodyEl = document.body;
  var stage, sheets, tbPage, tbZoom;
  var pages = [];
  var idToPage = {};
  var zoom = 1;
  var minZoom = 0.18, maxZoom = 4.2;
  var book = {
    title: bodyEl.getAttribute('data-book') || 'The Snow Programming Language',
    short: bodyEl.getAttribute('data-short') || 'Snow',
    home: bodyEl.getAttribute('data-home') || './',
    otherLabel: bodyEl.getAttribute('data-other-label') || '',
    otherURL: bodyEl.getAttribute('data-other-url') || ''
  };
  var flake = (book.home || './') + 'snowflake.svg';

  function makeMeasurer(width) {
    var m = document.createElement('div');
    m.setAttribute('aria-hidden', 'true');
    m.style.cssText = 'position:fixed;left:-99999px;top:0;width:' + (width || CW) + 'px;z-index:-1000;pointer-events:none;';
    document.body.appendChild(m);
    return m;
  }

  function collectAtoms() {
    var atoms = [];
    var ATOM = 'header.chap-head,h3,p,pre,table,ul,ol,figure.listing,figure,.note,.chapnav,.toc,.colophon,.cover';
    function labelOf(sec) {
      var k = sec.querySelector('.chap-kicker');
      var h = sec.querySelector('h2');
      var t = (k ? k.textContent.replace(/\s+/g, ' ').trim() : '');
      if (h) t = t + (t ? ' · ' : '') + h.textContent.replace(/\s+/g, ' ').trim();
      return t.toUpperCase();
    }
    function walk(node, label) {
      for (var i = 0; i < node.children.length; i++) {
        var el = node.children[i];
        if (el.classList.contains('chapter')) { walk(el, labelOf(el)); continue; }
        if (el.classList.contains('cover')) { atoms.push({ el: el, label: '', cover: true }); continue; }
        if (el.matches(ATOM)) { atoms.push({ el: el, label: label, cover: false }); continue; }
        walk(el, label);
      }
    }
    walk(matter, '');
    return atoms;
  }

  function measureStack(atoms) {
    var m = makeMeasurer();
    var tops = [], bottoms = [];
    atoms.forEach(function (a) {
      var c = a.el.cloneNode(true);
      c.removeAttribute('id');
      m.appendChild(c);
      var r = c.getBoundingClientRect();
      tops.push(r.top); bottoms.push(r.bottom);
    });
    var hs = atoms.map(function (a, i) {
      if (i + 1 < tops.length) return tops[i + 1] - tops[i];
      return bottoms[i] - tops[i];
    });
    document.body.removeChild(m);
    atoms.forEach(function (a, i) { a.h = hs[i]; });
    if (atoms.length && atoms[0].cover) atoms[0].h = CH;
  }

  function preLines(pre) {
    var t = pre.textContent;
    if (t.charAt(t.length - 1) === '\n') t = t.slice(0, -1);
    return t.split('\n');
  }

  function measureLineHeights(lines) {
    if (!lines.length) return [];
    var m = makeMeasurer();
    var p = document.createElement('pre');
    p.className = 'pme';
    m.appendChild(p);
    lines.forEach(function (ln) {
      var d = document.createElement('div');
      d.textContent = ln;
      p.appendChild(d);
    });
    document.body.removeChild(m);
    var hs = [];
    var cum = 0;
    var first = p.children[0] ? p.children[0].getBoundingClientRect().height : 0;
    for (var i = 0; i < p.children.length; i++) {
      var r = p.children[i].getBoundingClientRect();
      hs.push(r.height);
    }
    var avg = first || (hs.length ? hs[0] : 20);
    for (var j = 0; j < lines.length; j++) hs[j] = hs[j] || avg;
    return hs;
  }

  function measureHeight(el, width) {
    var m = makeMeasurer(width);
    var c = el.cloneNode(true);
    c.style.margin = '0';
    m.appendChild(c);
    document.body.removeChild(m);
    var r = c.getBoundingClientRect();
    return r.height;
  }

  function makePreChunk(lines) {
    var p = document.createElement('pre');
    p.textContent = lines.join('\n');
    return p;
  }

  function makeFigChunk(fig, lines, withCaption) {
    var f = fig.cloneNode(false);
    if (withCaption) {
      var cap = fig.querySelector('figcaption');
      if (cap) f.appendChild(cap.cloneNode(true));
    }
    f.appendChild(makePreChunk(lines));
    return f;
  }

  function buildPreChunks(node) {
    var fig = node.matches('figure.listing') ? node : null;
    var pre = fig ? node.querySelector('pre') : node;
    var lines = preLines(pre);
    var hs = measureLineHeights(lines);
    var chunks = [];
    var capH = 0, capEl = null;
    if (fig) {
      capEl = fig.querySelector('figcaption');
      capH = capEl ? measureHeight(capEl) : 0;
    }
    var i = 0;
    function take(avail, start) {
      var used = PRE_PAD;
      var t = [], k = start;
      for (; k < lines.length; k++) {
        if (used + hs[k] <= avail) { t.push(lines[k]); used += hs[k]; }
        else break;
      }
      if (!t.length && start < lines.length) { t.push(lines[start]); k = start + 1; }
      return { lines: t, next: k };
    }
    var first = take(CH, 0);
    chunks.push({ node: fig ? makeFigChunk(fig, first.lines, true) : makePreChunk(first.lines), h: (fig ? capH : 0) + PRE_PAD + first.lines.reduce(function (a, b, ix) { return a + hs[ix]; }, 0), keep: true });
    i = first.next;
    while (i < lines.length) {
      var t = take(CH - PRE_END_GAP, i);
      var partH = calculateChunkHeight(t.lines, hs.slice(i, i + t.lines.length));
      chunks.push({
        node: fig ? makePreChunk(t.lines) : makePreChunk(t.lines),
        h: PRE_PAD + partH,
        keep: true,
        fresh: true
      });
      i = t.next;
    }
    return chunks;
  }

  function calculateChunkHeight(t, lineHs) {
    var tot = 0;
    lineHs.forEach(function (x) { tot += x; });
    return tot;
  }

  function buildTableChunks(table) {
    var rows = Array.prototype.slice.call(table.querySelectorAll('tr'));
    if (!rows.length) return [{ node: table.cloneNode(true), h: measureHeight(table, CW), keep: true }];
    var head = rows[0] && rows[0].querySelector('th') ? rows[0] : null;
    var data = head ? rows.slice(1) : rows;
    var headH = head ? measureHeight(head, CW) : 0;
    var m = makeMeasurer();
    var rowHs = [];
    data.forEach(function (r) {
      var t = document.createElement('table');
      t.style.cssText = 'border-collapse:collapse;width:100%;';
      if (head) t.appendChild(head.cloneNode(true));
      t.appendChild(r.cloneNode(true));
      m.appendChild(t);
      rowHs.push(t.getBoundingClientRect().height - headH);
    });
    document.body.removeChild(m);
    var chunks = [];
    var start = 0;
    while (start < data.length) {
      var end = start, h = 0;
      while (end < data.length && h + rowHs[end] <= CH - PRE_END_GAP - headH) { h += rowHs[end]; end++; }
      if (end === start) { end = start + 1; h = rowHs[start]; }
      var t = table.cloneNode(false);
      if (head) t.appendChild(head.cloneNode(true));
      for (var k = start; k < end; k++) t.appendChild(data[k].cloneNode(true));
      chunks.push({ node: t, h: headH + h, keep: true, fresh: start > 0 });
      start = end;
    }
    return chunks;
  }

  function buildListChunks(list) {
    var items = Array.prototype.slice.call(list.children);
    if (!items.length) return [{ node: list.cloneNode(true), h: measureHeight(list, CW), keep: true }];
    var m = makeMeasurer();
    var hs = [];
    items.forEach(function (li) {
      var c = li.cloneNode(true);
      c.style.cssText = 'margin:0;';
      m.appendChild(c);
      var r = c.getBoundingClientRect();
      hs.push(r.height + 8);
    });
    document.body.removeChild(m);
    var chunks = [];
    var start = 0;
    while (start < items.length) {
      var end = start, h = 0;
      while (end < items.length && h + hs[end] <= CH - PRE_END_GAP) { h += hs[end]; end++; }
      if (end === start) { end = start + 1; h = hs[start]; }
      var nl = document.createElement(list.tagName.toLowerCase());
      nl.className = list.className;
      if (list.getAttribute('start')) nl.setAttribute('start', list.getAttribute('start'));
      for (var k = start; k < end; k++) nl.appendChild(items[k].cloneNode(true));
      chunks.push({ node: nl, h: h, keep: true, fresh: start > 0 });
      start = end;
    }
    return chunks;
  }

  function buildChunks(atom) {
    if (atom.el.tagName === 'PRE' || (atom.el.tagName === 'FIGURE' && atom.el.querySelector('pre'))) {
      return buildPreChunks(atom.el);
    }
    if (atom.el.tagName === 'TABLE') return buildTableChunks(atom.el);
    if (atom.el.tagName === 'UL' || atom.el.tagName === 'OL') return buildListChunks(atom.el);
    return [{ node: atom.el.cloneNode(true), h: atom.h || measureHeight(atom.el), keep: true }];
  }

  function paginate() {
    var atoms = collectAtoms();
    measureStack(atoms);
    pages = [];
    idToPage = {};
    var pg = { atoms: [], left: CH, label: '', cover: false };

    atoms.forEach(function (atom) {
      var chunks = buildChunks(atom);
      var first = true;
      chunks.forEach(function (chunk, ci) {
        if (pg.cover) { pg = newPage(''); }
        if (first && pg.left >= chunk.h) {
          placeChunk(pg, atom, chunk);
        } else {
          if (!first || chunk.h > 0) {
            pg = newPage(atom.label);
          }
          placeChunk(pg, atom, chunk);
          if (chunk.fresh) pg = newPage(atom.label);
        }
        first = false;
      });
    });

    if (!pages.length || (pages[pages.length - 1].atoms.length === 0)) pages.pop();

    function newPage(label) {
      var p = { atoms: [], left: CH, label: label || '', cover: false };
      pages.push(p);
      return p;
    }

    function placeChunk(pgx, atom, chunk) {
      var label = atom.label || pgx.label;
      if (atom.cover) {
        pgx.cover = true;
        pgx.label = 'Title Page';
        pgx.atoms.push({ el: chunk.node, cover: true, label: label });
        pgx.left = 0;
        return;
      }
      if (!pgx.label) pgx.label = label;
      chunk.node.setAttribute('data-page-height', '');
      pgx.atoms.push({ el: chunk.node });
      pgx.left -= chunk.h;
      var ids = atom.el.querySelectorAll('[id]');
      var pageIndex = pages.indexOf(pgx);
      for (var i = 0; i < ids.length; i++) idToPage[ids[i].id] = pageIndex;
    }
    render();
  }

  function render() {
    sheets.innerHTML = '';
    sheets.style.width = 'auto';
    pages.forEach(function (pg, i) {
      var pageEl = document.createElement('div');
      pageEl.className = 'page' + (pg.cover ? ' cover-page' : '');
      var head = document.createElement('div');
      head.className = 'page-headx';
      var h1 = document.createElement('span'); h1.textContent = book.title.toUpperCase();
      var h2 = document.createElement('span'); h2.textContent = pg.label;
      head.appendChild(h1); head.appendChild(h2);
      var body = document.createElement('div');
      body.className = 'page-body';
      (pg.atoms || []).forEach(function (a) { body.appendChild(a.el); });
      var foot = document.createElement('div');
      foot.className = 'page-footx';
      var pn = document.createElement('span'); pn.className = 'pnum'; pn.textContent = String(i + 1);
      foot.appendChild(pn);
      pageEl.appendChild(head); pageEl.appendChild(body); pageEl.appendChild(foot);
      sheets.appendChild(pageEl);
    });
    updateHeight();
  }

  function updateHeight() {
    var base = 46 + pages.length * (PH + 26) + sheetPadExtra();
    sheets.style.height = base * zoom + 'px';
    sheets.style.width = PW * zoom + 'px';
  }
  function sheetPadExtra() { return 0; }

  function buildToolbar() {
    var tb = document.createElement('div');
    tb.id = 'toolbar';
    var left = document.createElement('div');
    left.className = 'tb-group';
    var brand = document.createElement('a');
    brand.id = 'tbBrand';
    brand.href = book.home;
    var img = document.createElement('img');
    img.src = flake; img.alt = '';
    var span = document.createElement('span'); span.textContent = book.short;
    brand.appendChild(img); brand.appendChild(span);
    var title = document.createElement('span');
    title.id = 'tbTitle'; title.textContent = book.title;
    left.appendChild(brand);
    left.appendChild(sep());
    left.appendChild(title);
    var center = document.createElement('div');
    center.className = 'tb-group';
    var bPrev = btn('\u2039', 'Previous page'); bPrev.title = 'Previous page (\u2190)';
    var bNext = btn('\u203a', 'Next page'); bNext.title = 'Next page (\u2192)';
    var pageInfo = document.createElement('span'); pageInfo.id = 'tbPage';
    center.appendChild(bPrev); center.appendChild(pageInfo); center.appendChild(bNext);
    center.appendChild(sep());
    var bZo = btn('\u2212', 'Zoom out'); bZo.title = 'Zoom out (-)';
    var zoomInfo = document.createElement('span'); zoomInfo.id = 'tbZoom';
    var bZi = btn('+', 'Zoom in'); bZi.title = 'Zoom in (+)';
    var bFit = btn('Fit', 'Fit to width');
    var bFull = btn('\u2922', 'Fullscreen'); bFull.title = 'Fullscreen (f)';
    center.appendChild(bZo); center.appendChild(zoomInfo); center.appendChild(bZi);
    center.appendChild(sep());
    center.appendChild(bFit); center.appendChild(bFull);
    var right = document.createElement('div');
    right.className = 'tb-group';
    right.id = 'tbRight';
    if (book.otherLabel) {
      var oa = document.createElement('a');
      oa.href = book.otherURL; oa.textContent = book.otherLabel;
      right.appendChild(oa); right.appendChild(sep());
    }
    var gh = document.createElement('a');
    gh.href = (book.home || './') + '../';
    gh.textContent = '\u00A9 Snow';
    right.appendChild(gh);
    tb.appendChild(left);
    tb.appendChild(center);
    tb.appendChild(right);
    document.body.insertBefore(tb, document.body.firstChild);

    function sep() {
      var s = document.createElement('span'); s.className = 'tb-sep'; s.textContent = '\u00b7';
      return s;
    }
    function btn(label, tip) {
      var b = document.createElement('button');
      b.textContent = label; b.type = 'button';
      if (tip) b.setAttribute('aria-label', tip);
      return b;
    }
    tbPage = pageInfo; tbZoom = zoomInfo;
    bPrev.addEventListener('click', function () { setPage(-1, true); });
    bNext.addEventListener('click', function () { setPage(1, true); });
    bZo.addEventListener('click', function () { zoomBy(0.82); });
    bZi.addEventListener('click', function () { zoomBy(1.22); });
    bFit.addEventListener('click', function () { fitZoom(); });
    bFull.addEventListener('click', toggleFullscreen);
  }

  function setPage(i, rel) {
    var target = rel ? (cbPageNo() + i) : i;
    if (target < 0) target = 0;
    if (target >= pages.length) target = pages.length - 1;
    var pageEl = sheets.children[target];
    if (!pageEl) return;
    var zr = zoom;
    if (zr < 1) {
      stage.scrollTo({ top: (target === 0 ? 0 : target * (PH + 26) * zr - 10), left: stage.scrollLeft, behavior: 'smooth' });
    } else {
      stage.scrollTo({ top: (target === 0 ? 0 : target * (PH + 26)) - 10, left: stage.scrollLeft, behavior: 'smooth' });
    }
    updatePageInfo();
  }

  function cbPageNo() {
    if (!pages.length) return 0;
    var zr = zoom;
    var py = stage.scrollTop / zr;
    var idx = Math.round(py / (PH + 26));
    if (idx < 0) idx = 0;
    if (idx >= pages.length) idx = pages.length - 1;
    return idx;
  }

  function updatePageInfo() {
    if (tbPage) tbPage.textContent = (cbPageNo() + 1) + ' / ' + pages.length;
  }

  function applyZoom(z, keepCenter) {
    var z1 = zoom;
    if (z < minZoom) z = minZoom;
    if (z > maxZoom) z = maxZoom;
    zoom = z;
    sheets.style.transform = 'scale(' + z + ')';
    sheets.style.transformOrigin = '0 0';
    updateHeight();
    if (keepCenter && z1 > 0) {
      var s = z / z1;
      var ox = stage.scrollLeft + stage.clientWidth / 2 - sheets.offsetLeft;
      var oy = stage.scrollTop + stage.clientHeight / 2 - sheets.offsetTop;
      var nx = sheets.offsetLeft + ox * s - stage.clientWidth / 2;
      var ny = sheets.offsetTop + oy * s - stage.clientHeight / 2;
      stage.scrollLeft = nx;
      stage.scrollTop = ny;
    }
    updatePageInfo();
  }

  function zoomBy(f) {
    var targetY = stage.scrollTop + stage.clientHeight / 2;
    applyZoom(zoom * f, true);
  }

  function fitZoom() {
    var avail = stage.clientWidth - 84;
    var f = avail / PW;
    if (f < minZoom) f = minZoom;
    if (f > maxZoom) f = maxZoom;
    applyZoom(f, false);
    if (stage.scrollTop === 0) { stage.scrollLeft = 0; }
  }

  function toggleFullscreen() {
    if (document.fullscreenElement) {
      document.exitFullscreen && document.exitFullscreen();
    } else {
      var el = stage || document.documentElement;
      el.requestFullscreen && el.requestFullscreen().catch(function () {});
    }
  }

  function init() {
    buildToolbar();
    stage = document.getElementById('stage');
    sheets = document.getElementById('sheets');
    bodyEl.classList.add('paged');
    bodyEl.classList.add('jsenabled');
    var m = document.createElement('div');
    m.className = 'sheetpad';
    sheets.parentNode.insertBefore(m, sheets);
    paginate();
    fitZoom();
    window.addEventListener('resize', function () { fitZoom(); });
    document.addEventListener('keydown', function (e) {
      if (e.target && /INPUT|TEXTAREA/.test(e.target.tagName)) return;
      if (e.key === 'ArrowLeft' || e.key === 'PageUp') { e.preventDefault(); setPage(-1, true); }
      else if (e.key === 'ArrowRight' || e.key === 'PageDown') { e.preventDefault(); setPage(1, true); }
      else if (e.key === '+' || e.key === '=') { e.preventDefault(); zoomBy(1.22); }
      else if (e.key === '-' || e.key === '_') { e.preventDefault(); zoomBy(0.82); }
      else if (e.key === '0') { e.preventDefault(); fitZoom(); }
      else if (e.key === 'f' || e.key === 'F') { toggleFullscreen(); }
      else if (e.key === 'Escape' && document.fullscreenElement) { document.exitFullscreen(); }
    });
    document.addEventListener('wheel', function (e) {
      if (e.ctrlKey) { e.preventDefault(); zoomBy(e.deltaY < 0 ? 1.1 : 0.91); }
    }, { passive: false });
    sheets.addEventListener('click', function (e) {
      var a = e.target.closest ? e.target.closest('a[href^="#"]') : null;
      if (!a) return;
      var id = a.getAttribute('href').slice(1);
      if (idToPage[id] != null) { e.preventDefault(); setPage(idToPage[id], false); }
    });
  }

  function keepAlive() { }
  init();
  if (document.fonts && document.fonts.ready) {
    document.fonts.ready.then(function () {
      setTimeout(paginate, 40);
      updatePageInfo();
    });
  }
  window.addEventListener('load', function () { setTimeout(paginate, 60); });
})();