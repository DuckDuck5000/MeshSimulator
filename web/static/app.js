// File: web/static/app.js

document.addEventListener("DOMContentLoaded", async () => {
  // —— 1) Grab DOM elements —— 
  const modeExisting   = document.getElementById("modeExisting");
  const modeCustom     = document.getElementById("modeCustom");
  const existingBlock  = document.getElementById("existingBlock");
  const customBlock    = document.getElementById("customBlock");
  const topologySelect = document.getElementById("topology");
  const customTextarea = document.getElementById("customYaml");

  const tabPasteYaml   = document.getElementById("tabPasteYaml");
  const tabBuildForm   = document.getElementById("tabBuildForm");
  const pasteYamlBlock = document.getElementById("pasteYamlBlock");
  const buildFormBlock = document.getElementById("buildFormBlock");
  const nodeCount      = document.getElementById("nodeCount");
  const nodeSections   = document.getElementById("nodeSections");

  const dropRateSlider  = document.getElementById("dropRate");
  const dropRateValue   = document.getElementById("dropRateValue");
  const ttlSlider       = document.getElementById("ttl");
  const ttlValue        = document.getElementById("ttlValue");

  const loadBtn         = document.getElementById("loadTopology");
  const resetBtn        = document.getElementById("resetSim");
  const senderSelect    = document.getElementById("sender");
  const recipientSelect = document.getElementById("recipient");
  const sendBtn         = document.getElementById("sendMsg");

  const logDiv          = document.getElementById("log");
  const svg             = d3.select("#graph");
  let simulation;    // D3 force simulation
  let linkGroup, nodeGroup;

  // —— 2) Show/Hide “Existing” vs “Custom” blocks on load —— 
  if (modeExisting.checked) {
    existingBlock.style.display = "block";
    customBlock.style.display   = "none";
  } else {
    existingBlock.style.display = "none";
    customBlock.style.display   = "block";
  }
  modeExisting.addEventListener("change", () => {
    if (modeExisting.checked) {
      existingBlock.style.display = "block";
      customBlock.style.display   = "none";
      tabPasteYaml.checked        = true;
      pasteYamlBlock.style.display = "block";
      buildFormBlock.style.display = "none";
    }
  });
  modeCustom.addEventListener("change", () => {
    if (modeCustom.checked) {
      existingBlock.style.display = "none";
      customBlock.style.display   = "block";
      tabPasteYaml.checked        = true;
      pasteYamlBlock.style.display = "block";
      buildFormBlock.style.display = "none";
    }
  });

  // —— 3) Toggle “Paste YAML” vs “Build with Form” inside customBlock —— 
  if (tabPasteYaml.checked) {
    pasteYamlBlock.style.display = "block";
    buildFormBlock.style.display = "none";
  } else {
    pasteYamlBlock.style.display = "none";
    buildFormBlock.style.display = "block";
  }
  tabPasteYaml.addEventListener("change", () => {
    if (tabPasteYaml.checked) {
      pasteYamlBlock.style.display = "block";
      buildFormBlock.style.display = "none";
    }
  });
  tabBuildForm.addEventListener("change", () => {
    if (tabBuildForm.checked) {
      pasteYamlBlock.style.display = "none";
      buildFormBlock.style.display = "block";
    }
  });

  // —— 4) Populate the “Select Topology” dropdown from GET /topologies —— 
  try {
    const topoResponse = await fetch("/topologies");
    if (topoResponse.ok) {
      const topoList = await topoResponse.json();
      topoList.forEach((fname) => {
        const opt = document.createElement("option");
        opt.value = fname;
        opt.innerText = fname;
        topologySelect.appendChild(opt);
      });
    } else {
      console.error("GET /topologies →", topoResponse.status);
    }
  } catch (err) {
    console.error("Error fetching /topologies:", err);
  }

  // —— 5) Update Drop Rate & TTL slider labels —— 
  dropRateSlider.addEventListener("input", () => {
    dropRateValue.innerText = dropRateSlider.value;
  });
  ttlSlider.addEventListener("input", () => {
    ttlValue.innerText = ttlSlider.value;
  });

  // —— 6) Dynamically build “Node X” form sections when Number of Nodes changes —— 
  nodeCount.addEventListener("change", () => {
    nodeSections.innerHTML = "";
    const n = parseInt(nodeCount.value, 10);
    if (isNaN(n) || n < 1) return;

    const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";
    for (let i = 0; i < n; i++) {
      const defaultID = letters[i] || `N${i+1}`;
      const container = document.createElement("div");
      container.className = "node-section";

      // Node ID
      const lblId = document.createElement("label");
      lblId.innerText = `Node ${i+1} ID:`;
      const inputId = document.createElement("input");
      inputId.type = "text";
      inputId.value = defaultID;
      inputId.dataset.index = i;
      container.appendChild(lblId);
      container.appendChild(inputId);

      // Neighbors
      const lblNbr = document.createElement("label");
      lblNbr.innerText = `Neighbors for ${defaultID}:`;
      lblNbr.style.marginTop = "8px";
      container.appendChild(lblNbr);

      const cbContainer = document.createElement("div");
      cbContainer.style.marginLeft = "10px";
      for (let j = 0; j < n; j++) {
        if (j === i) continue;
        const nbID = letters[j] || `N${j+1}`;
        const cbDiv = document.createElement("div");
        cbDiv.className = "checkbox-row";   // use our CSS class
        const cb = document.createElement("input");
        cb.type = "checkbox";
        cb.value = nbID;
        cb.dataset.index = i;
        const cbLabel = document.createElement("label");
        cbLabel.innerText = nbID;
        cbLabel.style.margin = "0";
        cbDiv.appendChild(cb);
        cbDiv.appendChild(cbLabel);
        cbContainer.appendChild(cbDiv);
      }
      container.appendChild(cbContainer);
      nodeSections.appendChild(container);
    }
  });

  // —— 7) The “configureMesh” function, now with D3 2D rendering —— 
  async function configureMesh() {
    const payload = {
      dropRate: parseFloat(dropRateSlider.value),
      ttl:      parseInt(ttlSlider.value, 10),
    };

    // 7.a) Existing or Custom?
    if (modeExisting.checked) {
      const chosen = topologySelect.value;
      if (!chosen) {
        alert("Please select an existing topology first.");
        return;
      }
      payload.topology = chosen;
    } else {
      // Custom → Paste YAML or Build Form
      if (tabPasteYaml.checked) {
        const raw = customTextarea.value.trim();
        if (!raw) {
          alert("Paste your YAML first.");
          return;
        }
        const lineCount = (raw.match(/^- id:/gm) || []).length;
        if (lineCount > 30) {
          alert("Custom YAML may not exceed 30 nodes.");
          return;
        }
        payload.customYAML = raw;
      } else {
        const n = parseInt(nodeCount.value, 10);
        if (isNaN(n) || n < 1) {
          alert("Select the number of nodes first.");
          return;
        }
        const containers = nodeSections.querySelectorAll(".node-section");
        if (containers.length !== n) {
          alert("Please match the # of nodes with your form entries.");
          return;
        }
        const nodesArr = [];
        for (let i = 0; i < n; i++) {
          const cont = containers[i];
          const idField = cont.querySelector('input[type="text"]');
          const nid = idField.value.trim();
          if (!nid) {
            alert(`Node ${i+1} needs a valid ID.`);
            return;
          }
          const checkedBoxes = cont.querySelectorAll('input[type="checkbox"]:checked');
          const nbrs = Array.from(checkedBoxes).map(cb => cb.value);
          nodesArr.push({ id: nid, neighbors: nbrs });
        }
        if (nodesArr.length > 30) {
          alert("Topology exceed 30 nodes (limit).");
          return;
        }
        const yamlObj = { nodes: nodesArr };
        payload.customYAML = jsyaml.dump(yamlObj, { noRefs: true, indent: 2 });
      }
    }

    // 7.b) Clear old log & SVG
    logDiv.innerHTML = "";
    svg.selectAll("*").remove();     // clear previous graph

    // 7.c) POST /configure
    let configureResponse;
    try {
      configureResponse = await fetch("/configure", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
    } catch (err) {
      console.error("Network error POST /configure:", err);
      logEvent("Error: could not reach /configure.");
      return;
    }
    if (!configureResponse.ok) {
      const errText = await configureResponse.text();
      console.error("POST /configure →", configureResponse.status, errText);
      alert("Configuration failed: " + errText);
      return;
    }

    // 7.d) Log success of /configure
    const modeText = modeExisting.checked
      ? `existing "${payload.topology}"`
      : `"custom YAML" (built from form)`;
    logEvent(`Configured ${modeText}, dropRate=${payload.dropRate}, ttl=${payload.ttl}`);

    // 7.e) GET /nodes
    let nodesList;
    try {
      const nodesResp = await fetch("/nodes");
      if (!nodesResp.ok) {
        console.error("GET /nodes →", nodesResp.status);
        logEvent("Error: could not fetch node list.");
        return;
      }
      nodesList = await nodesResp.json();
    } catch (err) {
      console.error("Network error GET /nodes:", err);
      logEvent("Error: network problem getting nodes.");
      return;
    }

    // Populate Sender & Recipient dropdowns
    senderSelect.innerHTML    = "";
    recipientSelect.innerHTML = "";
    nodesList.forEach((nid) => {
      const o1 = document.createElement("option");
      o1.value = nid;
      o1.innerText = nid;
      senderSelect.appendChild(o1);

      const o2 = document.createElement("option");
      o2.value = nid;
      o2.innerText = nid;
      recipientSelect.appendChild(o2);
    });
    senderSelect.disabled    = false;
    recipientSelect.disabled = false;

    // 7.f) Fetch the YAML file or reuse custom
    let yamlText;
    if (modeExisting.checked) {
      const topoFile = topologySelect.value;
      let yamlResp;
      try {
        yamlResp = await fetch(`/topo-files/${encodeURIComponent(topoFile)}`);
      } catch (err) {
        console.error("Network error GET /topo-files:", err);
        logEvent("Error: could not fetch topology file.");
        return;
      }
      if (!yamlResp.ok) {
        console.error("GET /topo-files →", yamlResp.status);
        logEvent("Error: topology file not found.");
        return;
      }
      yamlText = await yamlResp.text();
    } else {
      if (tabPasteYaml.checked) {
        yamlText = customTextarea.value.trim();
      } else {
        // rebuild from form
        const n = parseInt(nodeCount.value, 10);
        const containers = nodeSections.querySelectorAll(".node-section");
        const nodesArr = [];
        for (let i = 0; i < n; i++) {
          const cont = containers[i];
          const nid = cont.querySelector('input[type="text"]').value.trim();
          const checkedBoxes = cont.querySelectorAll('input[type="checkbox"]:checked');
          const nbrs = Array.from(checkedBoxes).map(cb => cb.value);
          nodesArr.push({ id: nid, neighbors: nbrs });
        }
        yamlText = jsyaml.dump({ nodes: nodesArr }, { noRefs: true, indent: 2 });
      }
    }

    // 7.g) Parse YAML and build D3 nodes/links arrays
    let topoData;
    try {
      topoData = jsyaml.load(yamlText);
    } catch (err) {
      console.error("Error parsing YAML text:", err);
      logEvent("Error: invalid YAML.");
      return;
    }
    const nodesArr = topoData.nodes.map(n => ({ id: n.id }));
    const linksArr = [];
    topoData.nodes.forEach(nr => {
      nr.neighbors.forEach(nb => {
        if (nr.id < nb) {
          linksArr.push({ source: nr.id, target: nb });
        }
      });
    });

    // 7.h) Render the D3 force‐directed graph
    render2DGraph(nodesArr, linksArr);

    // 7.i) Enable Send & Reset
    sendBtn.disabled  = false;
    resetBtn.disabled = false;
  }

  // 8) Wire up “Load Topology” & “Reset Simulation”
  loadBtn.addEventListener("click", configureMesh);
  resetBtn.addEventListener("click", configureMesh);

  // 9) “Send Message” logic
  sendBtn.addEventListener("click", async () => {
    const from = senderSelect.value;
    const to   = recipientSelect.value;
    if (!from || !to) {
      alert("Select both sender and recipient.");
      return;
    }
    try {
      const resp = await fetch("/send", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ from, to }),
      });
      if (!resp.ok) {
        console.error("POST /send →", resp.status);
        logEvent("Error: message not queued.");
      } else {
        logEvent(`Queued message from ${from} → ${to}`);
      }
    } catch (err) {
      console.error("Network error POST /send:", err);
      logEvent("Error sending message.");
    }
  });

  // 10) WebSocket for real‐time events
  const protocol = window.location.protocol === "https:" ? "wss" : "ws";
  const wsUrl    = `${protocol}://${window.location.host}/ws`;
  const socket   = new WebSocket(wsUrl);
  socket.addEventListener("open", () => {
    logEvent("WebSocket connected.");
  });
  socket.addEventListener("close", () => {
    logEvent("WebSocket disconnected.");
  });
  socket.addEventListener("message", (evt) => {
    try {
      const ev = JSON.parse(evt.data);
      handleEvent(ev);
    } catch (err) {
      console.error("Invalid event JSON:", err);
    }
  });

  // 11) Logging helper
  function logEvent(text) {
    const p = document.createElement("div");
    p.innerText = `[${new Date().toLocaleTimeString()}] ${text}`;
    logDiv.appendChild(p);
    logDiv.scrollTop = logDiv.scrollHeight;
  }

  // 12) Handle incoming WebSocket events
  function handleEvent(ev) {
    switch (ev.type) {
      case "received":
        logEvent(`Node ${ev.to} RECEIVED from ${ev.from} (TTL=${ev.ttl})`);
        break;
      case "dropped_ttl":
        logEvent(`Node ${ev.to} DROPPED_TTL from ${ev.from}`);
        break;
      case "decrypt_failed":
        logEvent(`Node ${ev.to} DECRYPT_FAILED from ${ev.from}: ${ev.payload.error}`);
        break;
      case "decrypted":
        logEvent(`Node ${ev.to} DECRYPTED from ${ev.from}: "${ev.payload.plaintext}"`);
        break;
      case "forwarded":
        logEvent(`Node ${ev.from} FORWARDED (new TTL=${ev.ttl})`);
        break;
      case "dropped_network":
        logEvent(`Network DROPPED between ${ev.from} → ${ev.to}`);
        break;
      case "delivered":
        logEvent(`Network DELIVERED from ${ev.from} → ${ev.to} (TTL=${ev.ttl})`);
        animateDot(ev.from, ev.to);
        break;
      default:
        logEvent(`Unknown event type: ${JSON.stringify(ev)}`);
    }
  }

  // ────────────────── 2D Visualization with D3.js ──────────────────

  function render2DGraph(nodesData, linksData) {
    // Clear any existing SVG contents
    svg.selectAll("*").remove();

        const container = document.getElementById("vis-container");
    const width = container.clientWidth;
    const height = container.clientHeight;

    // Set SVG dimensions
    svg
        .attr("width", width)
        .attr("height", height)
        .attr("viewBox", `0 0 ${width} ${height}`);

    // Create D3 force simulation
    simulation = d3.forceSimulation(nodesData)
      .force("link", d3.forceLink(linksData).id(d => d.id).distance(80))
      .force("charge", d3.forceManyBody().strength(-300))
      .force("center", d3.forceCenter(width / 2, height / 2));

    // Draw links (lines)
    linkGroup = svg.append("g")
      .attr("class", "links")
      .selectAll("line")
      .data(linksData)
      .join("line")
      .attr("class", "link-line");

    // Draw nodes (circles + labels)
    nodeGroup = svg.append("g")
      .attr("class", "nodes")
      .selectAll("g")
      .data(nodesData)
      .join("g")
      .call(drag(simulation)); // allow dragging

    nodeGroup.append("circle")
      .attr("class", "node-circle")
      .attr("r", 15)
      .attr("fill", "#2a9df4");

    nodeGroup.append("text")
      .attr("class", "node-label")
      .attr("dy", 4)
      .text(d => d.id);

    // On each tick, update positions
    simulation.on("tick", () => {
      linkGroup
        .attr("x1", d => d.source.x)
        .attr("y1", d => d.source.y)
        .attr("x2", d => d.target.x)
        .attr("y2", d => d.target.y);

      nodeGroup
        .attr("transform", d => `translate(${d.x},${d.y})`);
    });
  }

  // D3 drag behavior
  function drag(sim) {
    function dragstarted(event, d) {
      if (!event.active) sim.alphaTarget(0.3).restart();
      d.fx = d.x;
      d.fy = d.y;
    }

    function dragged(event, d) {
      d.fx = event.x;
      d.fy = event.y;
    }

    function dragended(event, d) {
      if (!event.active) sim.alphaTarget(0);
      d.fx = null;
      d.fy = null;
    }

    return d3.drag()
      .on("start", dragstarted)
      .on("drag", dragged)
      .on("end", dragended);
  }

  // 13) Animate a small red “dot” traveling from one node to another
  function animateDot(fromID, toID) {
    const srcData = simulation.nodes().find(n => n.id === fromID);
    const tgtData = simulation.nodes().find(n => n.id === toID);
    if (!srcData || !tgtData) return;

    const dot = svg.append("circle")
      .attr("class", "packet-dot")
      .attr("r", 6)
      .attr("cx", srcData.x)
      .attr("cy", srcData.y)
      .attr("opacity", 0.8);

    dot.transition()
      .duration(1000)
      .attr("cx", tgtData.x)
      .attr("cy", tgtData.y)
      .attr("opacity", 0)
      .remove();
  }

    window.addEventListener("resize", () => {
    if (!simulation) return;
    const container = document.getElementById("vis-container");
    const w = container.clientWidth;
    const h = container.clientHeight;
    
    svg
        .attr("width", w)
        .attr("height", h)
        .attr("viewBox", `0 0 ${w} ${h}`);
        
    simulation
        .force("center", d3.forceCenter(w / 2, h / 2))
        .alpha(0.3)
        .restart();
    });
});
