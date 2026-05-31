// E-COMMERCE HUB dashboard client.
(() => {
  "use strict";

  const AGENTS = ["BENEFIT", "PROMO", "DESIGN", "PROMPT", "STUDIO", "QUALITY CHECK"];
  const SLUG = { "BENEFIT": "benefit", "PROMO": "promo", "DESIGN": "design",
                 "PROMPT": "prompt", "STUDIO": "studio", "QUALITY CHECK": "qc" };

  let token = localStorage.getItem("ecom_token") || "";
  let evtSource = null;
  let chart = null;
  let teamFilter = "all";
  const baht = (n) => "฿" + (n || 0).toLocaleString();

  // ---------- DOM helpers ----------
  const $ = (id) => document.getElementById(id);
  const api = (path, opts = {}) => {
    opts.headers = Object.assign({ "Content-Type": "application/json" }, opts.headers || {});
    if (token) opts.headers["Authorization"] = "Bearer " + token;
    return fetch(path, opts);
  };

  // ---------- Auth ----------
  async function authRequest(path) {
    const email = $("authEmail").value.trim();
    const password = $("authPassword").value;
    $("authErr").textContent = "";
    const r = await fetch(path, {
      method: "POST", headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });
    const data = await r.json();
    if (!r.ok) { $("authErr").textContent = data.error || "เกิดข้อผิดพลาด"; return; }
    token = data.token;
    localStorage.setItem("ecom_token", token);
    enterApp();
  }

  function logout() {
    localStorage.removeItem("ecom_token");
    token = "";
    if (evtSource) evtSource.close();
    location.reload();
  }

  // ---------- App lifecycle ----------
  function enterApp() {
    $("authOverlay").classList.add("hidden");
    $("app").style.visibility = "visible";
    $("kpiBar").style.visibility = "visible";
    renderAgentCards();
    loadDashboard();
    connectSSE();
  }

  async function loadDashboard() {
    const r = await api("/api/v1/dashboard");
    if (r.status === 401) { logout(); return; }
    const d = await r.json();
    renderStats(d);
    renderOrders(d.order_summary);
    renderProducts(d.listings || []);
    renderChart(d.sales_series || []);
    renderQuota(d);
    renderTeam(d.listings || []);
    const me = await (await api("/api/v1/me")).json();
    if (me.user) $("userBadge").textContent = me.user.email;
  }

  // ---------- Renderers ----------
  function renderStats(d) {
    const s = d.stats;
    $("kSales").textContent = baht(s.overall_sales_thb);
    $("kOrders").textContent = s.total_orders.toLocaleString();
    $("kUnits").textContent = s.units_sold.toLocaleString();
    $("kProducts").textContent = s.active_products.toLocaleString();
    $("kRating").textContent = s.shop_rating;
    $("kResponse").textContent = s.response_rate + "%";
    $("tTotalSales").textContent = baht(s.overall_sales_thb);
    $("tVisitors").textContent = (s.units_sold * 5).toLocaleString();
    $("tConv").textContent = s.active_products ? "3.28%" : "0%";
  }

  function renderOrders(o) {
    if (!o) return;
    $("oTotal").textContent = o.total.toLocaleString();
    $("oPending").textContent = o.pending;
    $("oShipped").textContent = o.shipped;
    $("oCancelled").textContent = o.cancelled;
    $("oAvg").textContent = baht(o.avg_order_value);
  }

  function renderQuota(d) {
    if (d.plan) $("planName").textContent = (d.plan.name || "FREE").toUpperCase();
    if (d.quota) $("quotaUsage").textContent = `${d.quota.used} / ${d.quota.limit}`;
  }

  function renderProducts(listings) {
    const grid = $("productGrid");
    grid.innerHTML = "";
    if (!listings.length) {
      $("emptyHint").textContent = 'ยังไม่มีสินค้า — กด "+ ADD PRODUCT" ให้ AI สร้าง listing แรกของคุณ!';
      return;
    }
    $("emptyHint").textContent = "";
    listings.forEach((j) => {
      const res = j.result || {};
      const img = (res.image_url) || "";
      const price = (j.id.charCodeAt(0) % 15 + 3) * 100 + 99; // deterministic display price
      const card = document.createElement("div");
      card.className = "product-card";
      const stClass = j.status === "done" ? "" : j.status;
      card.innerHTML = `
        ${res.promotion ? `<span class="badge">PROMO</span>` : ""}
        <span class="badge status ${stClass}">${j.status.toUpperCase()}</span>
        <img class="thumb" src="${img}" alt="" onerror="this.style.opacity=.25">
        <div class="pname" title="${escapeHtml(j.product_name)}">${escapeHtml(j.product_name)}</div>
        <div class="price">฿${price.toLocaleString()}</div>
        <div class="stock">${res.headline ? escapeHtml(truncate(res.headline, 22)) : "กำลังประมวลผล..."}</div>
      `;
      grid.appendChild(card);
    });
  }

  function renderChart(series) {
    const ctx = $("salesChart").getContext("2d");
    const labels = series.map((_, i) => "D" + (i + 1));
    if (chart) { chart.data.labels = labels; chart.data.datasets[0].data = series; chart.update(); return; }
    chart = new Chart(ctx, {
      type: "line",
      data: { labels, datasets: [{
        data: series, borderColor: "#ff6a2d", backgroundColor: "rgba(255,106,45,.15)",
        fill: true, tension: .35, pointBackgroundColor: "#ffd23f", pointRadius: 3,
      }]},
      options: {
        plugins: { legend: { display: false } },
        scales: {
          x: { ticks: { color: "#6f86b8" }, grid: { color: "#16264c" } },
          y: { ticks: { color: "#6f86b8" }, grid: { color: "#16264c" } },
        },
      },
    });
  }

  // Agent work cards (right column) + team rows share live state.
  const agentState = {}; // name -> {percent, task}

  function renderAgentCards() {
    const wrap = $("agentCards");
    wrap.innerHTML = "";
    AGENTS.forEach((name) => {
      const slug = SLUG[name];
      const card = document.createElement("div");
      card.className = "work-card";
      card.id = "wc-" + slug;
      card.innerHTML = `
        <div class="wc-head"><span class="wc-name">${name}</span><span class="status-dot" id="dot-${slug}"></span></div>
        <img class="wc-art" src="/static/img/agent-${slug}.png" alt="" onerror="this.style.opacity=.2">
        <div class="wc-task" id="task-${slug}">IDLE <span class="wc-pct" id="pct-${slug}">0%</span></div>
        <div class="bar cyan"><span id="bar-${slug}"></span></div>
      `;
      wrap.appendChild(card);
    });
  }

  function renderTeam(listings) {
    const active = listings.filter((j) => j.status === "running" || j.status === "pending").length;
    const list = $("teamList");
    list.innerHTML = "";
    AGENTS.forEach((name) => {
      const slug = SLUG[name];
      const st = agentState[name] || { percent: 0, task: "IDLE" };
      const working = st.percent > 0 && st.percent < 100;
      if (teamFilter === "working" && !working) return;
      if (teamFilter === "idle" && working) return;
      const row = document.createElement("div");
      row.className = "agent-row";
      row.innerHTML = `
        <div class="agent-avatar" style="background-image:url('/static/img/agent-${slug}.png')"></div>
        <div class="agent-meta">
          <div class="agent-name">${name}</div>
          <div class="agent-state ${working ? "working" : ""}">[${working ? "WORKING" : "IDLE"}]</div>
          <div class="bar"><span style="width:${st.percent}%"></span></div>
        </div>
        <div class="status-dot ${working ? "on" : ""}"></div>
      `;
      list.appendChild(row);
    });
    if (!list.children.length) list.innerHTML = `<p class="muted" style="font-size:15px">— ไม่มี agent ในสถานะนี้ —</p>`;
    void active;
  }

  // ---------- SSE ----------
  function connectSSE() {
    if (evtSource) evtSource.close();
    evtSource = new EventSource("/api/v1/events?token=" + encodeURIComponent(token));
    evtSource.addEventListener("progress", (e) => {
      const ev = JSON.parse(e.data);
      updateAgent(ev.agent, ev.percent, ev.task);
    });
    evtSource.addEventListener("job_done", () => { resetAgents(); loadDashboard(); toast("✅ สร้าง listing เสร็จแล้ว!"); });
    evtSource.addEventListener("job_failed", (e) => {
      const ev = JSON.parse(e.data);
      resetAgents(); toast("⚠️ งานล้มเหลว: " + (ev.task || ""), true); loadDashboard();
    });
  }

  function updateAgent(name, percent, task) {
    if (!name || !SLUG[name]) return;
    agentState[name] = { percent, task };
    const slug = SLUG[name];
    const bar = $("bar-" + slug), pct = $("pct-" + slug), tk = $("task-" + slug), dot = $("dot-" + slug);
    if (bar) bar.style.width = percent + "%";
    if (pct) pct.textContent = percent + "%";
    if (tk) tk.innerHTML = `${escapeHtml(task || "")} <span class="wc-pct" id="pct-${slug}">${percent}%</span>`;
    if (dot) dot.classList.toggle("on", percent > 0 && percent < 100);
    renderTeam(window.__lastListings || []);
  }

  function resetAgents() {
    AGENTS.forEach((n) => agentState[n] = { percent: 0, task: "IDLE" });
    AGENTS.forEach((n) => {
      const slug = SLUG[n];
      if ($("bar-" + slug)) $("bar-" + slug).style.width = "0%";
      if ($("pct-" + slug)) $("pct-" + slug).textContent = "0%";
      if ($("task-" + slug)) $("task-" + slug).innerHTML = `IDLE <span class="wc-pct">0%</span>`;
      if ($("dot-" + slug)) $("dot-" + slug).classList.remove("on");
    });
  }

  // ---------- Create listing ----------
  async function generate() {
    const product_name = $("newProduct").value.trim();
    const lang = $("newLang").value;
    $("addErr").textContent = "";
    if (!product_name) { $("addErr").textContent = "กรุณาใส่ชื่อสินค้า"; return; }
    const r = await api("/api/v1/listings", { method: "POST", body: JSON.stringify({ product_name, lang }) });
    const data = await r.json();
    if (r.status === 402) { $("addErr").textContent = "โควต้าเดือนนี้เต็มแล้ว — อัปเกรดแพ็กเกจเพื่อสร้างเพิ่ม"; return; }
    if (!r.ok) { $("addErr").textContent = data.error || "สร้างไม่สำเร็จ"; return; }
    $("addOverlay").classList.add("hidden");
    $("newProduct").value = "";
    resetAgents();
    toast("🚀 AI กำลังทำงาน...");
    loadDashboard();
  }

  // ---------- Upgrade ----------
  async function upgrade(plan) {
    const r = await api("/api/v1/billing/checkout", { method: "POST", body: JSON.stringify({ plan }) });
    const data = await r.json();
    if (data.url) { window.location.href = data.url; }
    else { toast(data.error || "เปิด checkout ไม่ได้", true); }
  }

  // ---------- Misc UI ----------
  function clock() {
    const d = new Date();
    let h = d.getHours(); const m = String(d.getMinutes()).padStart(2, "0");
    const ap = h >= 12 ? "PM" : "AM"; h = h % 12 || 12;
    $("clock").textContent = `${String(h).padStart(2, "0")}:${m} ${ap}`;
  }
  function toast(msg, bad) {
    const t = document.createElement("div");
    t.className = "toast"; t.textContent = msg;
    if (bad) t.style.borderColor = "#ff4d5e", t.style.color = "#ff4d5e";
    document.body.appendChild(t);
    setTimeout(() => t.remove(), 3500);
  }
  const escapeHtml = (s) => (s || "").replace(/[&<>"']/g, (c) =>
    ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
  const truncate = (s, n) => (s && s.length > n ? s.slice(0, n) + "…" : s);

  // ---------- Wire events ----------
  $("btnLogin").onclick = () => authRequest("/api/v1/auth/login");
  $("btnRegister").onclick = () => authRequest("/api/v1/auth/register");
  $("btnLogout").onclick = logout;
  $("btnAdd").onclick = () => $("addOverlay").classList.remove("hidden");
  $("btnCancelAdd").onclick = () => $("addOverlay").classList.add("hidden");
  $("btnGenerate").onclick = generate;
  document.querySelectorAll(".upgrade-btn").forEach((b) => b.onclick = () => upgrade(b.dataset.plan));
  document.querySelectorAll(".filters .chip").forEach((c) => c.onclick = () => {
    document.querySelectorAll(".filters .chip").forEach((x) => x.classList.remove("active"));
    c.classList.add("active"); teamFilter = c.dataset.f; renderTeam([]);
  });
  document.addEventListener("keydown", (e) => { if (e.key === "Enter" && !$("authOverlay").classList.contains("hidden")) authRequest("/api/v1/auth/login"); });

  setInterval(clock, 1000); clock();

  // Auto-enter if we already have a token.
  if (token) enterApp();
})();
