(function () {
  "use strict";
  var data = window.PLUME_DATA || { nodes: [], flows: [], categories: [] };
  var elk = new ELK();
  var NS = "http://www.w3.org/2000/svg";

  var nodeById = {}, catById = {};
  data.nodes.forEach(function (n) { nodeById[n.id] = n; });
  (data.categories || []).forEach(function (c) { catById[c.id] = c; });

  var KIND = {
    source: { fill: "#06281f", label: "Source" }, service: { fill: "#0e2138", label: "Service" },
    store: { fill: "#2a1f0c", label: "Store" }, sink: { fill: "#22262e", label: "Sink" },
    external: { fill: "#2c1116", label: "External" }
  };
  var SENS_RANK = { health: 4, credential: 4, special: 4, financial: 3, pii: 2, public: 1 };
  function css(v) { return getComputedStyle(document.documentElement).getPropertyValue(v).trim() || "#8aa0b8"; }
  function kindColor(k) { return css("--" + k); }
  function rank(s) { return SENS_RANK[s] || 0; }
  function catSens(id) { return (catById[id] || {}).sensitivity || "public"; }
  function flowSens(f) { var s = "public"; (f.categories || []).forEach(function (c) { if (rank(catSens(c)) > rank(s)) s = catSens(c); }); return s; }
  function catLabels(f) { return (f.categories || []).map(function (c) { return (catById[c] || {}).label || c; }); }

  var out = {}, inc = {};
  data.nodes.forEach(function (n) { out[n.id] = []; inc[n.id] = []; });
  data.flows.forEach(function (f) { (out[f.from] || (out[f.from] = [])).push(f.to); (inc[f.to] || (inc[f.to] = [])).push(f.from); });
  function lineage(id) {
    var set = {}; set[id] = true;
    (function up(x) { (inc[x] || []).forEach(function (p) { if (!set[p]) { set[p] = true; up(p); } }); })(id);
    (function dn(x) { (out[x] || []).forEach(function (p) { if (!set[p]) { set[p] = true; dn(p); } }); })(id);
    return set;
  }

  var activeSens = {}; Object.keys(SENS_RANK).forEach(function (s) { activeSens[s] = true; });
  var focusSet = null, searchTerm = "", viewMode = "graph";

  function visibleFlows() {
    return data.flows.filter(function (f) {
      var cs = (f.categories && f.categories.length) ? f.categories.map(catSens) : ["public"];
      return cs.some(function (s) { return activeSens[s]; });
    });
  }

  var svg = document.getElementById("svg");
  var root = document.getElementById("root");
  var tip = document.getElementById("tip");
  var emptyEl = document.getElementById("empty");
  function el(name, attrs) { var e = document.createElementNS(NS, name); if (attrs) for (var k in attrs) e.setAttribute(k, attrs[k]); return e; }

  var defs = el("defs");
  var marker = el("marker", { id: "arrow", viewBox: "0 0 10 10", refX: "9", refY: "5", markerWidth: "7", markerHeight: "7", orient: "auto-start-reverse" });
  marker.appendChild(el("path", { d: "M0 0L10 5L0 10z", fill: "#5b7390" }));
  defs.appendChild(marker); svg.insertBefore(defs, root);

  function nodeW(n) { var len = (n.label || n.id).length; return Math.max(104, Math.min(248, 24 + len * 8)); }

  // ---------- graph view ----------
  function renderGraph() {
    var flows = visibleFlows();
    var used = {}; flows.forEach(function (f) { used[f.from] = true; used[f.to] = true; });
    var nodes = data.nodes.filter(function (n) { return used[n.id]; });
    emptyEl.style.display = flows.length ? "none" : "flex";
    if (!flows.length) { root.innerHTML = ""; return; }
    var g = {
      id: "root",
      layoutOptions: {
        "elk.algorithm": "layered", "elk.direction": "RIGHT", "elk.edgeRouting": "ORTHOGONAL",
        "elk.layered.spacing.nodeNodeBetweenLayers": "96", "elk.spacing.nodeNode": "26",
        "elk.layered.spacing.edgeNodeBetweenLayers": "22"
      },
      children: nodes.map(function (n) { return { id: n.id, width: nodeW(n), height: 46 }; }),
      edges: flows.map(function (f, i) { return { id: "e" + i, sources: [f.from], targets: [f.to] }; })
    };
    elk.layout(g).then(function (res) { paint(res, nodes, flows); }).catch(function (err) { console.error("ELK", err); });
  }

  function paint(res, nodes, flows) {
    root.innerHTML = "";
    var pos = {}; (res.children || []).forEach(function (c) { pos[c.id] = c; });
    var eLayer = el("g"); root.appendChild(eLayer);
    (res.edges || []).forEach(function (e) {
      var f = flows[parseInt(e.id.slice(1), 10)]; if (!f) return;
      var sec = (e.sections || [])[0]; if (!sec) return;
      var pts = [sec.startPoint].concat(sec.bendPoints || [], [sec.endPoint]);
      var d = "M" + pts.map(function (p) { return p.x + " " + p.y; }).join(" L ");
      var grp = el("g", { "class": "edge", "data-from": f.from, "data-to": f.to });
      grp.appendChild(el("path", { d: d, stroke: css("--" + flowSens(f)), "marker-end": "url(#arrow)" }));
      var labels = catLabels(f);
      if (labels.length) {
        var mid = pts[Math.floor(pts.length / 2)] || pts[0];
        var t = el("text", { x: mid.x, y: mid.y - 5, "text-anchor": "middle" });
        t.textContent = labels.join(", "); grp.appendChild(t);
      }
      grp.addEventListener("mousemove", function (ev) { showTip(ev, edgeTip(f)); });
      grp.addEventListener("mouseleave", hideTip);
      eLayer.appendChild(grp);
    });
    nodes.forEach(function (n) {
      var p = pos[n.id]; if (!p) return;
      var grp = el("g", { "class": "node", transform: "translate(" + p.x + "," + p.y + ")", "data-id": n.id });
      grp.appendChild(el("rect", { width: p.width, height: p.height, rx: 9, fill: (KIND[n.kind] || {}).fill || "#16202e", stroke: kindColor(n.kind), "stroke-width": 1.5 }));
      var label = el("text", { x: 12, y: 20 }); label.textContent = n.label || n.id; grp.appendChild(label);
      var meta = el("text", { x: 12, y: 36, "class": "meta" }); meta.textContent = (KIND[n.kind] || {}).label || n.kind; grp.appendChild(meta);
      grp.addEventListener("click", function (ev) { ev.stopPropagation(); setFocus(focusSet && focusSet[n.id] && focusOnly === n.id ? null : n.id); });
      grp.addEventListener("mousemove", function (ev) { showTip(ev, nodeTip(n)); });
      grp.addEventListener("mouseleave", hideTip);
      root.appendChild(grp);
    });
    applyHighlight(); fit();
  }

  // ---------- sankey view ----------
  function renderSankey() {
    root.innerHTML = "";
    var flows = visibleFlows();
    emptyEl.style.display = flows.length ? "none" : "flex";
    if (!flows.length) return;
    var used = {}; flows.forEach(function (f) { used[f.from] = true; used[f.to] = true; });
    var nodes = data.nodes.filter(function (n) { return used[n.id]; });
    // depth = longest path from a node with no incoming visible edge
    var vin = {}, vout = {};
    nodes.forEach(function (n) { vin[n.id] = []; vout[n.id] = []; });
    flows.forEach(function (f) { vout[f.from].push(f); vin[f.to].push(f); });
    var depth = {};
    function dof(id, seen) { if (depth[id] != null) return depth[id]; if (seen[id]) return 0; seen[id] = true;
      var d = 0; vin[id].forEach(function (f) { d = Math.max(d, dof(f.from, seen) + 1); }); depth[id] = d; return d; }
    nodes.forEach(function (n) { dof(n.id, {}); });
    var layers = {}; nodes.forEach(function (n) { (layers[depth[n.id]] || (layers[depth[n.id]] = [])).push(n); });
    var maxL = Math.max.apply(null, Object.keys(layers).map(Number));
    var colW = 230, H = 560, nodeH = 26, pad = 16;
    var weight = {}; nodes.forEach(function (n) {
      var w = 0; vin[n.id].concat(vout[n.id]).forEach(function (f) { w += Math.max(1, (f.categories || []).length); });
      weight[n.id] = Math.max(1, w);
    });
    var box = {};
    Object.keys(layers).forEach(function (L) {
      var arr = layers[L]; var total = 0; arr.forEach(function (n) { total += Math.max(nodeH, weight[n.id] * 10); });
      var y = pad, x = Number(L) * colW + pad;
      arr.forEach(function (n) { var h = Math.max(nodeH, weight[n.id] * 10); box[n.id] = { x: x, y: y, w: 150, h: h }; y += h + 14; });
    });
    var eLayer = el("g"); root.appendChild(eLayer);
    flows.forEach(function (f) {
      var a = box[f.from], b = box[f.to]; if (!a || !b) return;
      var x1 = a.x + a.w, y1 = a.y + a.h / 2, x2 = b.x, y2 = b.y + b.h / 2, mx = (x1 + x2) / 2;
      var w = Math.max(2, Math.min(18, (f.categories || []).length * 6));
      var grp = el("g", { "class": "edge", "data-from": f.from, "data-to": f.to });
      grp.appendChild(el("path", { d: "M" + x1 + " " + y1 + " C " + mx + " " + y1 + " " + mx + " " + y2 + " " + x2 + " " + y2,
        stroke: css("--" + flowSens(f)), "stroke-width": w, "stroke-opacity": .45, fill: "none" }));
      grp.addEventListener("mousemove", function (ev) { showTip(ev, edgeTip(f)); });
      grp.addEventListener("mouseleave", hideTip);
      eLayer.appendChild(grp);
    });
    nodes.forEach(function (n) {
      var b = box[n.id]; var grp = el("g", { "class": "node", transform: "translate(" + b.x + "," + b.y + ")", "data-id": n.id });
      grp.appendChild(el("rect", { width: b.w, height: b.h, rx: 5, fill: (KIND[n.kind] || {}).fill || "#16202e", stroke: kindColor(n.kind), "stroke-width": 1.5 }));
      var t = el("text", { x: 9, y: b.h / 2 + 4 }); t.textContent = n.label || n.id; grp.appendChild(t);
      grp.addEventListener("click", function (ev) { ev.stopPropagation(); setFocus(focusOnly === n.id ? null : n.id); });
      grp.addEventListener("mousemove", function (ev) { showTip(ev, nodeTip(n)); });
      grp.addEventListener("mouseleave", hideTip);
      root.appendChild(grp);
    });
    applyHighlight(); fit();
  }

  function rerender() { if (viewMode === "graph") renderGraph(); else renderSankey(); }

  // ---------- focus + search ----------
  var focusOnly = null;
  function setFocus(id) { focusOnly = id; focusSet = id ? lineage(id) : null; applyHighlight(); }
  function applyHighlight() {
    var nodesEls = root.querySelectorAll(".node");
    var edgeEls = root.querySelectorAll(".edge");
    for (var i = 0; i < nodesEls.length; i++) {
      var id = nodesEls[i].getAttribute("data-id");
      var inFocus = !focusSet || focusSet[id];
      var inSearch = !searchTerm || ((nodeById[id].label || id).toLowerCase().indexOf(searchTerm) >= 0);
      nodesEls[i].classList.toggle("faded", !(inFocus && inSearch));
    }
    for (var j = 0; j < edgeEls.length; j++) {
      var fr = edgeEls[j].getAttribute("data-from"), to = edgeEls[j].getAttribute("data-to");
      var vis = !focusSet || (focusSet[fr] && focusSet[to]);
      edgeEls[j].classList.toggle("faded", !vis);
    }
  }

  // ---------- tooltip ----------
  function nodeTip(n) {
    var s = "<b>" + esc(n.label || n.id) + "</b><div class='k'>" + ((KIND[n.kind] || {}).label || n.kind) + (n.system ? " · " + esc(n.system) : "") + "</div>";
    if (n.location) s += "<div class='k'>" + esc(n.location) + "</div>";
    return s;
  }
  function edgeTip(f) {
    var s = "<b>" + esc(nm(f.from)) + " → " + esc(nm(f.to)) + "</b>";
    var labs = catLabels(f); if (labs.length) s += "<div>" + esc(labs.join(", ")) + "</div>";
    if (f.via) s += "<div class='k'>via " + esc(f.via) + "</div>";
    if (f.evidence) s += "<div class='k'>" + esc(f.evidence) + "</div>";
    return s;
  }
  function nm(id) { return (nodeById[id] || {}).label || id; }
  function esc(s) { return String(s).replace(/[&<>]/g, function (c) { return { "&": "&amp;", "<": "&lt;", ">": "&gt;" }[c]; }); }
  function showTip(ev, html) { tip.innerHTML = html; tip.style.display = "block"; tip.style.left = (ev.clientX + 14) + "px"; tip.style.top = (ev.clientY + 14) + "px"; }
  function hideTip() { tip.style.display = "none"; }

  // ---------- pan / zoom / fit ----------
  var tx = 0, ty = 0, scale = 1;
  function apply() { root.setAttribute("transform", "translate(" + tx + "," + ty + ") scale(" + scale + ")"); }
  function fit() {
    var bb; try { bb = root.getBBox(); } catch (e) { return; }
    if (!bb.width) return;
    var vw = svg.clientWidth, vh = svg.clientHeight, m = 40;
    scale = Math.min((vw - m) / bb.width, (vh - m) / bb.height, 1.4);
    tx = (vw - bb.width * scale) / 2 - bb.x * scale;
    ty = (vh - bb.height * scale) / 2 - bb.y * scale;
    apply();
  }
  svg.addEventListener("wheel", function (ev) {
    ev.preventDefault();
    var f = ev.deltaY < 0 ? 1.1 : 1 / 1.1, mx = ev.clientX, my = ev.clientY;
    tx = mx - (mx - tx) * f; ty = my - (my - ty) * f; scale *= f; apply();
  }, { passive: false });
  var drag = null;
  svg.addEventListener("mousedown", function (ev) { drag = { x: ev.clientX - tx, y: ev.clientY - ty }; svg.classList.add("dragging"); });
  window.addEventListener("mousemove", function (ev) { if (drag) { tx = ev.clientX - drag.x; ty = ev.clientY - drag.y; apply(); } });
  window.addEventListener("mouseup", function () { drag = null; svg.classList.remove("dragging"); });
  svg.addEventListener("click", function () { if (focusOnly) setFocus(null); });

  // ---------- chrome ----------
  function buildFilters() {
    var wrap = document.getElementById("filters");
    var order = ["pii", "financial", "credential", "health", "special", "public"];
    var present = {}; (data.categories || []).forEach(function (c) { present[c.sensitivity || "public"] = true; });
    order.filter(function (s) { return present[s]; }).forEach(function (s) {
      var c = document.createElement("div"); c.className = "chip on";
      c.innerHTML = "<span class='dot' style='background:" + css("--" + s) + "'></span>" + s;
      c.onclick = function () { activeSens[s] = !activeSens[s]; c.classList.toggle("on", activeSens[s]); c.classList.toggle("off", !activeSens[s]); rerender(); };
      wrap.appendChild(c);
    });
  }
  function buildLegend() {
    var l = document.getElementById("legend"), html = "";
    [["source", "Source (user)"], ["service", "Service (code)"], ["store", "Data store"], ["sink", "Sink (logs)"], ["external", "Third party"]].forEach(function (p) {
      html += "<div class='row'><span class='sw' style='border-color:" + css("--" + p[0]) + ";background:" + (KIND[p[0]] || {}).fill + "'></span>" + p[1] + "</div>";
    });
    l.innerHTML = html;
  }
  document.getElementById("counts").textContent =
    data.nodes.length + " nodes · " + data.flows.length + " flows";
  document.getElementById("search").addEventListener("input", function (e) { searchTerm = e.target.value.toLowerCase(); applyHighlight(); });
  document.getElementById("fit").addEventListener("click", fit);
  var vg = document.getElementById("vGraph"), vs = document.getElementById("vSankey");
  vg.onclick = function () { viewMode = "graph"; vg.classList.add("on"); vs.classList.remove("on"); rerender(); };
  vs.onclick = function () { viewMode = "sankey"; vs.classList.add("on"); vg.classList.remove("on"); rerender(); };

  buildFilters(); buildLegend(); rerender();
  window.addEventListener("resize", fit);
})();
