const form = document.querySelector("#applicationForm");
const verifyButton = document.querySelector("#verifyButton");
const statusText = document.querySelector("#statusText");
const verdictBadge = document.querySelector("#verdictBadge");
const fieldsBody = document.querySelector("#fieldsBody");
const confidenceText = document.querySelector("#confidenceText");
const singleRunStamp = document.querySelector("#singleRunStamp");
const summaryStrip = document.querySelector("#summaryStrip");
const rawJson = document.querySelector("#rawJson");
const imageInput = document.querySelector("#labelImage");
const imagePreview = document.querySelector("#imagePreview");
const previewFrame = document.querySelector(".preview-frame");
const fileName = document.querySelector("#fileName");
const derivedApplicationId = document.querySelector("#derivedApplicationId");
const singleFixtureSelect = document.querySelector("#singleFixtureSelect");
const loadSingleFixtureButton = document.querySelector("#loadSingleFixtureButton");
const applicationSaveStatus = document.querySelector("#applicationSaveStatus");
const judgeRunStatus = document.querySelector("#judgeRunStatus");
const decisionStatus = document.querySelector("#decisionStatus");
const decisionRuntime = document.querySelector("#decisionRuntime");
const decisionFailed = document.querySelector("#decisionFailed");
const decisionRisk = document.querySelector("#decisionRisk");
const decisionJudge = document.querySelector("#decisionJudge");
const fieldDetail = document.querySelector("#fieldDetail");
const detailBody = document.querySelector("#detailBody");
const failedFieldsPanel = document.querySelector("#failedFieldsPanel");
const failedFieldsCount = document.querySelector("#failedFieldsCount");
const failedFieldsList = document.querySelector("#failedFieldsList");
const manageAppsButton = document.querySelector("#manageAppsButton");
const manageModal = document.querySelector("#manageModal");
const manageModalClose = document.querySelector("#manageModalClose");
const manageList = document.querySelector("#manageList");
const manageSelectAll = document.querySelector("#manageSelectAll");
const manageSelectionCount = document.querySelector("#manageSelectionCount");
const manageDeleteSelected = document.querySelector("#manageDeleteSelected");
const singleModeButton = document.querySelector("#singleModeButton");
const batchModeButton = document.querySelector("#batchModeButton");
const singleView = document.querySelector("#singleView");
const batchView = document.querySelector("#batchView");
const loadBatchFixturesButton = document.querySelector("#loadBatchFixturesButton");
const addBatchRowButton = document.querySelector("#addBatchRowButton");
const runBatchButton = document.querySelector("#runBatchButton");
const batchRowsBody = document.querySelector("#batchRowsBody");
const batchConcurrency = document.querySelector("#batchConcurrency");
const batchStatus = document.querySelector("#batchStatus");
const batchCount = document.querySelector("#batchCount");
const batchConsistent = document.querySelector("#batchConsistent");
const batchFlagged = document.querySelector("#batchFlagged");
const batchErrors = document.querySelector("#batchErrors");
const batchRunStamp = document.querySelector("#batchRunStamp");
const batchResultsBody = document.querySelector("#batchResultsBody");
const batchRawJson = document.querySelector("#batchRawJson");
const downloadBatchButton = document.querySelector("#downloadBatchButton");
const toastHost = document.querySelector("#toastHost");

let currentFields = [];
let selectedFieldIndex = -1;
let batchRowCounter = 0;
let lastBatchResponse = null;
let singleRunCounter = 0;
let batchRunCounter = 0;

const fixtures = [
  {
    label: "Generated Rye Whiskey",
    id: "generated-rye-whiskey",
    image: "generated_rye_whiskey.png",
    application: {
      Brand: "POM CREEK DISTILLING COMPANY, LLC",
      ClassType: "Rye Whiskey",
      NetContents: "750 mL",
      ABV: "45% ALC/VOL (90 Proof)",
      GovernmentWarning: "GOVERNMENT WARNING",
      NameAndAddress: "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA",
      ForeignBlocks: [],
      DeclaredLanguages: ["en"],
    },
  },
  {
    label: "Generated Blueberry Liqueur",
    id: "generated-blueberry-liqueur",
    image: "generated_blueberry_liqueur.png",
    application: {
      Brand: "POM CREEK DISTILLING COMPANY, LLC",
      ClassType: "Blueberry Liqueur/Cordial",
      NetContents: "375 mL",
      ABV: "25% ALC/VOL (50 Proof)",
      GovernmentWarning: "GOVERNMENT WARNING",
      NameAndAddress: "POM CREEK DISTILLING COMPANY, LLC LOUDOUN COUNTY, VA",
      ForeignBlocks: [],
      DeclaredLanguages: ["en"],
    },
  },
  {
    label: "Generated Apple Brandy",
    id: "generated-apple-brandy",
    image: "generated_apple_brandy.png",
    application: {
      Brand: "POM CREEK DISTILLING COMPANY, LLC",
      ClassType: "Apple Brandy",
      NetContents: "750 mL",
      ABV: "40% ALC/VOL (80 Proof)",
      GovernmentWarning: "GOVERNMENT WARNING",
      NameAndAddress: "POM CREEK DISTILLING COMPANY, LLC LOUDOUN COUNTY, VA",
      ForeignBlocks: [],
      DeclaredLanguages: ["en"],
    },
  },
  {
    label: "Generated Bourbon With Contents",
    id: "generated-bourbon-with-contents",
    image: "generated_bourbon_with_contents.png",
    application: {
      Brand: "POM BOURBON",
      ClassType: "Bourbon",
      NetContents: "750 mL",
      ABV: "45% ALC/VOL (90 Proof)",
      GovernmentWarning: "GOVERNMENT WARNING",
      NameAndAddress: "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA",
      ForeignBlocks: [],
      DeclaredLanguages: ["en"],
    },
  },
  {
    label: "Generated Bourbon No Contents",
    id: "generated-bourbon-no-contents",
    image: "generated_bourbon_no_contents.png",
    application: {
      Brand: "POM BOURBON",
      ClassType: "Bourbon",
      NetContents: "750 mL",
      ABV: "45% ALC/VOL (90 Proof)",
      GovernmentWarning: "GOVERNMENT WARNING",
      NameAndAddress: "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA",
      ForeignBlocks: [],
      DeclaredLanguages: ["en"],
    },
  },
  {
    label: "Generated Bourbon Government Warning No Contents",
    id: "generated-bourbon-gov-warning-no-contents",
    image: "generated_bourbon_gov_warning_no_contents.png",
    application: {
      Brand: "POM BOURBON",
      ClassType: "Bourbon",
      NetContents: "750 mL",
      ABV: "45% ALC/VOL (90 Proof)",
      GovernmentWarning: "GOVERNMENT WARNING",
      NameAndAddress: "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA",
      ForeignBlocks: [],
      DeclaredLanguages: ["en"],
    },
  },
  {
    label: "Generated Bourbon Government Warning With Contents",
    id: "generated-bourbon-gov-warning-with-contents",
    image: "generated_bourbon_gov_warning_with_contents.png",
    application: {
      Brand: "POM BOURBON",
      ClassType: "Bourbon",
      NetContents: "750 mL",
      ABV: "45% ALC/VOL (90 Proof)",
      GovernmentWarning: "GOVERNMENT WARNING",
      NameAndAddress: "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA",
      ForeignBlocks: [],
      DeclaredLanguages: ["en"],
    },
  },
  {
    label: "Generated Bourbon No Government Warning",
    id: "generated-bourbon-no-gov-warning",
    image: "generated_bourbon_no_gov_warning.jpg",
    application: {
      Brand: "POM BOURBON",
      ClassType: "Bourbon",
      NetContents: "750 mL",
      ABV: "45% ALC/VOL (90 Proof)",
      GovernmentWarning: "GOVERNMENT WARNING",
      NameAndAddress: "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA",
      ForeignBlocks: [],
      DeclaredLanguages: ["en"],
    },
  },
  {
    label: "Generated Tequila",
    id: "generate-tequila",
    image: "generate-tequila.png",
    application: {
      Brand: "POM BLANCO TEQUILA",
      ClassType: "Blanco Tequila",
      NetContents: "750 mL",
      ABV: "40% ALC/VOL (80 Proof)",
      GovernmentWarning: "GOVERNMENT WARNING",
      NameAndAddress: "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA",
      ForeignBlocks: [],
      DeclaredLanguages: ["en"],
    },
  },
];

imageInput.addEventListener("change", handleManualImageSelection);

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  await saveApplication();
});

form.addEventListener(
  "invalid",
  () => {
    setApplicationSaveStatus("Complete all required application fields", "error");
    setStatus("Complete all required application fields");
  },
  true,
);

verifyButton.addEventListener("click", () => runSingleVerification().catch(handleSingleError));

async function runSingleVerification() {
  await saveApplication();
  await verifyLabel();
}

singleModeButton.addEventListener("click", () => setMode("single"));
batchModeButton.addEventListener("click", () => setMode("batch"));
singleFixtureSelect.addEventListener("change", handleFixtureSelectionChange);
loadSingleFixtureButton.addEventListener("click", loadSelectedSingleFixture);
loadBatchFixturesButton.addEventListener("click", loadFixtureBatch);
addBatchRowButton.addEventListener("click", () => addBatchRow());
runBatchButton.addEventListener("click", () => runBatch().catch(handleBatchError));
downloadBatchButton.addEventListener("click", downloadBatchResults);
document.querySelector("#judgeEnabled").addEventListener("change", updateJudgeHint);
document.querySelector("#judgeMode").addEventListener("change", updateJudgeHint);
document.querySelectorAll('[data-judge-field="single"]').forEach((input) => {
  input.addEventListener("change", updateJudgeHint);
});

populateFixtureOptions();
updateJudgeHint();
for (let i = 0; i < 3; i++) addBatchRow();

function setMode(mode) {
  const batch = mode === "batch";
  singleView.hidden = batch;
  batchView.hidden = !batch;
  singleModeButton.classList.toggle("active", !batch);
  batchModeButton.classList.toggle("active", batch);
  setStatus(batch ? "Batch mode ready" : "Single label mode ready");
}

function populateFixtureOptions() {
  singleFixtureSelect.innerHTML = `<option value="">Custom upload</option>` + fixtures
    .map((fixture) => `<option value="${escapeHtml(fixture.id)}">${escapeHtml(fixture.label)}</option>`)
    .join("");
  singleFixtureSelect.value = "";
}

async function loadSelectedSingleFixture() {
  const fixture = fixtureByID(singleFixtureSelect.value);
  if (!fixture) {
    setApplicationSaveStatus("", "");
    setStatus("Select an example fixture to load");
    return;
  }

  setBusy(true, `Loading ${fixture.label}`);
  try {
    singleFixtureSelect.value = fixture.id;
    applyFixtureToForm(fixture);
    const file = await loadFixtureImage(fixture);
    assignFileToInput(imageInput, file);
    updateSingleImageUI(file);
    setBusy(false, `Loaded ${fixture.label}`);
  } catch (error) {
    setBusy(false, error.message || "Could not load example");
  }
}

function handleFixtureSelectionChange() {
  if (singleFixtureSelect.value) {
    setApplicationSaveStatus("", "");
    setStatus("Example selected. Click Load Example to populate the form.");
    return;
  }
  resetCustomUploadState();
}

function resetCustomUploadState() {
  imageInput.value = "";
  clearApplicationForm();
  updateSingleImageUI(null);
  setApplicationSaveStatus("", "");
  resetVerificationOutput();
  setStatus("Custom upload ready. Choose an image and enter application data.");
}

function handleManualImageSelection() {
  const file = imageInput.files?.[0];
  singleFixtureSelect.value = "";
  clearApplicationForm();
  updateSingleImageUI(file);
  setApplicationSaveStatus(file ? "Enter application data for this image" : "", "");
  setStatus(file ? "Custom image loaded. Complete the application fields before saving." : "Ready");
}

async function loadFixtureBatch() {
  setMode("batch");
  setBatchBusy(true, "Loading fixture batch");
  batchRowsBody.innerHTML = "";
  batchRowCounter = 0;
  lastBatchResponse = null;
  batchRawJson.textContent = "";
  batchResultsBody.innerHTML = `<tr><td colspan="5" class="empty">No batch run yet</td></tr>`;
  downloadBatchButton.disabled = true;

  try {
    for (const fixture of fixtures) {
      await saveFixtureApplication(fixture);
      const row = addBatchRow({ id: fixture.id, applicationId: fixture.id });
      const file = await loadFixtureImage(fixture);
      setBatchRowFile(row, file);
    }
    batchStatus.textContent = "Ready";
    batchCount.textContent = `${fixtures.length} / 100`;
    batchConsistent.textContent = "--";
    batchFlagged.textContent = "--";
    batchErrors.textContent = "--";
    if (batchRunStamp) batchRunStamp.textContent = "No batch run yet";
    setBatchBusy(false, `Loaded ${fixtures.length} fixture records`);
  } catch (error) {
    setBatchBusy(false, error.message || "Could not load fixture batch");
  }
}

async function saveApplication() {
  const file = imageInput.files?.[0];
  const id = file ? applicationIDFromFile(file) : "";
  if (!id) {
    setApplicationSaveStatus("Choose an image first", "error");
    throw setStatus("Choose a label image to generate the application ID", "error");
  }
  if (!validateApplicationForm()) {
    throw setStatus("Complete all required application fields", "error");
  }

  setApplicationSaveStatus("Saving...", "");
  setBusy(true, "Saving application");
  const response = await fetch(`/api/applications/${encodeURIComponent(id)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(applicationPayload(id)),
  });
  if (!response.ok) {
    const message = await response.text();
    setApplicationSaveStatus("Save failed", "error");
    setBusy(false, "Save failed");
    throw new Error(message.trim() || "Save failed");
  }
  setApplicationSaveStatus("Application saved", "success");
  setBusy(false, "Application saved");
}

async function verifyLabel() {
  const file = imageInput.files?.[0];
  if (!file) {
    setStatus("Choose a label image", "error");
    return;
  }
  const applicationID = applicationIDFromFile(file);
  if (document.querySelector("#judgeEnabled").checked && selectedJudgeFields("single").length === 0) {
    setJudgeRunStatus(`Select at least one field for AI ${reviewNoun(value("judgeMode"))}.`);
    setStatus("Select at least one AI reader field", "error");
    return;
  }

  const body = new FormData();
  body.set("application_id", applicationID);
  body.set("image", file);
  body.set("judge_enabled", document.querySelector("#judgeEnabled").checked ? "true" : "false");
  body.set("judge_mode", value("judgeMode"));
  body.set("judge_timeout_ms", value("judgeTimeoutMs"));
  body.set("judge_allowed_fields", selectedJudgeFields("single").join(","));

  const judgeRequested = document.querySelector("#judgeEnabled").checked;
  setJudgeRunStatus(
    judgeRequested
      ? `OCR/matching and AI label reading are running in parallel. Selected failed fields can be ${reviewPastTense(value("judgeMode"))}: ${selectedJudgeFieldLabels("single")}.`
      : "AI Label Reader is off for this run.",
  );

  setBusy(true, "Running verification");
  const started = performance.now();
  const response = await fetch("/api/verify", { method: "POST", body });
  const elapsed = Math.round(performance.now() - started);
  if (!response.ok) {
    const message = await response.text();
    setBusy(false, "Verification failed");
    throw new Error(message.trim() || "Verification failed");
  }

  const verdict = await response.json();
  const run = nextRunMeta("single");
  renderVerdict(verdict, elapsed, run);
  const message = completionMessage(verdict, elapsed);
  setBusy(false, message);
  notifyRun(`Run ${run.number} ${verdict.Status || "completed"}`, `${run.time} · ${message}`, verdict.Status === "consistent" ? "success" : "warning");
}

function addBatchRow(values = {}) {
  const count = batchRowsBody.querySelectorAll(".batch-input-row").length;
  if (count >= 100) {
    setStatus("Batch limit is 100 rows");
    return;
  }
  batchRowCounter += 1;
  const rowID = values.id || `row-${batchRowCounter}`;
  const tr = document.createElement("tr");
  tr.className = "batch-input-row";
  tr.dataset.rowId = rowID;
  tr.innerHTML = `
    <td><span class="batch-row-number">${count + 1}</span></td>
    <td><strong class="batch-derived-id">${escapeHtml(values.applicationId || "Choose image")}</strong></td>
    <td>
      <label class="batch-file-label">
        <input class="batch-file-input" type="file" accept="image/png,image/jpeg,image/webp" />
        <span>Choose image</span>
      </label>
    </td>
    <td><span class="batch-row-status idle">Queued</span></td>
    <td><button type="button" class="small-button danger batch-remove">Remove</button></td>`;
  batchRowsBody.appendChild(tr);

  const fileInput = tr.querySelector(".batch-file-input");
  const fileLabel = tr.querySelector(".batch-file-label span");
  const derivedID = tr.querySelector(".batch-derived-id");
  fileInput.addEventListener("change", () => {
    const file = fileInput.files?.[0];
    fileLabel.textContent = file?.name || "Choose image";
    derivedID.textContent = file ? applicationIDFromFile(file) : "Choose image";
  });
  tr.querySelector(".batch-remove").addEventListener("click", () => {
    tr.remove();
    updateBatchRowNumbers();
  });
  updateBatchRowNumbers();
  return tr;
}

function updateBatchRowNumbers() {
  const rows = batchRowsBody.querySelectorAll(".batch-input-row");
  rows.forEach((row, index) => {
    row.querySelector(".batch-row-number").textContent = String(index + 1);
  });
  addBatchRowButton.disabled = rows.length >= 100;
}

async function runBatch() {
  const rows = Array.from(batchRowsBody.querySelectorAll(".batch-input-row"));
  if (!rows.length) {
    setStatus("Add at least one batch row");
    return;
  }
  if (rows.length > 100) {
    setStatus("Batch limit is 100 rows");
    return;
  }
  if (document.querySelector("#batchJudgeEnabled").checked && selectedJudgeFields("batch").length === 0) {
    setStatus("Select at least one batch AI reader field", "error");
    return;
  }

  const items = [];
  const seenIDs = new Set();
  let valid = true;

  rows.forEach((row, index) => {
    const file = row.querySelector(".batch-file-input").files?.[0];
    const applicationID = file ? applicationIDFromFile(file) : "";
    const id = applicationID;
    const status = row.querySelector(".batch-row-status");
    status.textContent = "Queued";
    status.className = "batch-row-status idle";
    if (!file || !applicationID || seenIDs.has(id)) {
      status.textContent = seenIDs.has(id) ? "Duplicate ID" : "Incomplete";
      status.className = "batch-row-status error";
      valid = false;
      return;
    }
    seenIDs.add(id);
    items.push({ id, application_id: applicationID, file, row });
  });

  if (!valid) {
    setStatus("Fix incomplete or duplicate batch rows");
    return;
  }

  setBatchBusy(true, `Running batch of ${items.length}`);
  rows.forEach((row) => {
    const status = row.querySelector(".batch-row-status");
    status.textContent = "Running";
    status.className = "batch-row-status running";
  });

  const started = performance.now();
  const results = await runBatchItems(items, boundedClientBatchConcurrency(batchConcurrency.value));
  const elapsed = Math.round(performance.now() - started);
  const payload = buildClientBatchPayload(results, elapsed);
  lastBatchResponse = payload;
  const run = nextRunMeta("batch");
  renderBatchResults(payload, elapsed, run);
  setBatchBusy(false, `Batch completed in ${elapsed} ms`);
  notifyRun(`Batch ${run.number} completed`, `${run.time} · ${payload.summary?.consistent || 0} consistent, ${payload.summary?.flagged || 0} flagged, ${payload.summary?.errors || 0} errors`, payload.summary?.errors ? "warning" : "success");
}

async function runBatchItems(items, concurrency) {
  const results = new Array(items.length);
  let next = 0;
  const workerCount = Math.min(concurrency, items.length);
  await Promise.all(
    Array.from({ length: workerCount }, async () => {
      while (next < items.length) {
        const index = next;
        next += 1;
        results[index] = await verifyBatchItem(items[index]);
      }
    }),
  );
  return results;
}

async function verifyBatchItem(item) {
  const started = performance.now();
  const status = item.row.querySelector(".batch-row-status");
  status.textContent = "Running";
  status.className = "batch-row-status running";

  const body = new FormData();
  body.set("application_id", item.application_id);
  body.set("image", item.file);
  body.set("judge_enabled", document.querySelector("#batchJudgeEnabled").checked ? "true" : "false");
  body.set("judge_mode", value("batchJudgeMode"));
  body.set("judge_timeout_ms", value("batchJudgeTimeoutMs"));
  body.set("judge_allowed_fields", selectedJudgeFields("batch").join(","));

  try {
    const response = await fetch("/api/verify", { method: "POST", body });
    const durationMS = Math.round(performance.now() - started);
    if (!response.ok) {
      const message = await response.text();
      return {
        id: item.id,
        application_id: item.application_id,
        status: "error",
        duration_ms: durationMS,
        error: friendlyVerificationError(new Error(message.trim()), "Verification failed"),
      };
    }
    const verdict = await response.json();
    return {
      id: item.id,
      application_id: item.application_id,
      status: String(verdict.Status || "").toLowerCase() || "completed",
      duration_ms: durationMS,
      verdict,
    };
  } catch (error) {
    return {
      id: item.id,
      application_id: item.application_id,
      status: "error",
      duration_ms: Math.round(performance.now() - started),
      error: friendlyVerificationError(error, "Verification failed"),
    };
  }
}

function buildClientBatchPayload(results, elapsed) {
  const summary = results.reduce(
    (acc, result) => {
      if (result.status === "error") acc.errors += 1;
      else if (result.status === "consistent") acc.consistent += 1;
      else acc.flagged += 1;
      return acc;
    },
    { consistent: 0, flagged: 0, errors: 0 },
  );
  return {
    status: summary.errors ? "completed_with_errors" : "completed",
    count: results.length,
    limit: 100,
    duration_ms: elapsed,
    summary,
    results,
  };
}

function boundedClientBatchConcurrency(value) {
  const parsed = Number.parseInt(value || "1", 10);
  if (!Number.isFinite(parsed) || parsed < 1) return 1;
  return Math.min(parsed, 8);
}

function renderBatchResults(payload, elapsed, run) {
  const results = payload.results || [];
  batchStatus.textContent = payload.status || "completed";
  if (batchRunStamp && run) {
    batchRunStamp.textContent = `Batch ${run.number} completed at ${run.time}`;
  }
  batchCount.textContent = `${payload.count || results.length} / ${payload.limit || 100}`;
  batchConsistent.textContent = String(payload.summary?.consistent || 0);
  batchFlagged.textContent = String(payload.summary?.flagged || 0);
  batchErrors.textContent = String(payload.summary?.errors || 0);
  batchRawJson.textContent = JSON.stringify(payload, null, 2);
  downloadBatchButton.disabled = false;

  const rowsByID = new Map(
    Array.from(batchRowsBody.querySelectorAll(".batch-input-row")).map((row) => [
      row.querySelector(".batch-derived-id")?.textContent.trim() || row.dataset.rowId || "",
      row,
    ]),
  );
  results.forEach((result) => {
    const row = rowsByID.get(result.id);
    if (!row) return;
    const status = row.querySelector(".batch-row-status");
    if (!status) return;
    status.textContent = result.status || "done";
    status.className = `batch-row-status ${result.status || "idle"}`;
  });

  batchResultsBody.innerHTML = results.length
    ? results
        .map((result) => {
          const failed = result.verdict?.Fields?.filter((field) => !field.Pass) || [];
          const failedText = result.error
            ? result.error
            : failed.length
              ? failed.map((field) => `${field.Field}: ${actionLabel(field)}`).join(", ")
              : "None";
          return `
            <tr>
              <td>${escapeHtml(result.id || "")}</td>
              <td>${escapeHtml(result.application_id || "")}</td>
              <td><span class="batch-row-status ${escapeHtml(result.status || "idle")}">${escapeHtml(result.status || "unknown")}</span></td>
              <td>${formatDuration(result.duration_ms)}</td>
              <td>${escapeHtml(failedText)}</td>
            </tr>`;
        })
        .join("")
    : `<tr><td colspan="5" class="empty">No results returned</td></tr>`;

  batchStatus.textContent = `${payload.status || "completed"} · ${elapsed} ms`;
}

function handleBatchError(error) {
  const message = friendlyVerificationError(error, "Batch failed");
  setBatchBusy(false, message);
  notifyRun("Batch failed", message, "error");
  batchRowsBody.querySelectorAll(".batch-row-status.running").forEach((status) => {
    status.textContent = "Error";
    status.className = "batch-row-status error";
  });
}

function handleSingleError(error) {
  const message = friendlyVerificationError(error, "Verification failed");
  setBusy(false, message);
  notifyRun("Verification failed", message, "error");
  if (/timed out|deadline/i.test(message)) {
    setJudgeRunStatus("Verification timed out before the full result could be saved. Try running without AI Reader or reduce the reader timeout.");
  }
}

function friendlyVerificationError(error, fallback) {
  const raw = (error?.message || fallback || "Request failed").trim();
  if (/context deadline exceeded|verification timed out/i.test(raw)) {
    return "Verification timed out before the request budget completed.";
  }
  return raw;
}

function setBatchBusy(isBusy, message) {
  runBatchButton.disabled = isBusy;
  addBatchRowButton.disabled = isBusy || batchRowsBody.querySelectorAll(".batch-input-row").length >= 100;
  loadBatchFixturesButton.disabled = isBusy;
  downloadBatchButton.disabled = isBusy || !lastBatchResponse;
  setStatus(message);
}

function nextRunMeta(kind) {
  const now = new Date();
  const number = kind === "batch" ? ++batchRunCounter : ++singleRunCounter;
  return {
    number,
    time: now.toLocaleTimeString([], { hour: "numeric", minute: "2-digit", second: "2-digit" }),
  };
}

function notifyRun(title, message, tone = "info") {
  if (!toastHost) return;
  const toast = document.createElement("div");
  toast.className = `toast ${tone}`;
  toast.setAttribute("role", tone === "error" ? "alert" : "status");
  toast.innerHTML = `
    <strong>${escapeHtml(title)}</strong>
    <span>${escapeHtml(message)}</span>
  `;
  toastHost.appendChild(toast);
  window.setTimeout(() => {
    toast.classList.add("leaving");
    window.setTimeout(() => toast.remove(), 220);
  }, 5200);
}

function downloadBatchResults() {
  if (!lastBatchResponse) return;
  const blob = new Blob([JSON.stringify(lastBatchResponse, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `batch-results-${new Date().toISOString().replaceAll(":", "-")}.json`;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

function applicationPayload(id) {
  return {
    ID: id,
    Brand: value("brand"),
    ClassType: value("classType"),
    NetContents: value("netContents"),
    ABV: value("abv"),
    GovernmentWarning: value("governmentWarning"),
    NameAndAddress: value("nameAddress"),
    ForeignBlocks: [],
    DeclaredLanguages: ["en"],
  };
}

function applyFixtureToForm(fixture) {
  const app = fixture.application;
  setValue("brand", app.Brand);
  setValue("classType", app.ClassType);
  setValue("netContents", app.NetContents);
  setValue("abv", app.ABV);
  setValue("governmentWarning", app.GovernmentWarning);
  setValue("nameAddress", app.NameAndAddress);
}

function clearApplicationForm() {
  ["brand", "classType", "netContents", "abv", "governmentWarning", "nameAddress"].forEach((id) => setValue(id, ""));
}

function validateApplicationForm() {
  const required = [
    ["brand", "Brand"],
    ["classType", "Class / Type"],
    ["netContents", "Net Contents"],
    ["abv", "Alcohol Content"],
    ["nameAddress", "Name and Address"],
    ["governmentWarning", "Government Warning"],
  ];
  const missing = required.filter(([id]) => !value(id)).map(([, label]) => label);
  if (missing.length) {
    setApplicationSaveStatus(`Required: ${missing.join(", ")}`, "error");
    document.querySelector(`#${required.find(([id]) => !value(id))[0]}`)?.focus();
    form.reportValidity();
    return false;
  }
  if (!form.reportValidity()) {
    setApplicationSaveStatus("Complete required application fields", "error");
    return false;
  }
  return true;
}

function fixtureApplicationPayload(fixture) {
  return {
    ID: fixture.id,
    Brand: fixture.application.Brand,
    ClassType: fixture.application.ClassType,
    NetContents: fixture.application.NetContents,
    ABV: fixture.application.ABV,
    GovernmentWarning: fixture.application.GovernmentWarning,
    NameAndAddress: fixture.application.NameAndAddress,
    ForeignBlocks: fixture.application.ForeignBlocks || [],
    DeclaredLanguages: fixture.application.DeclaredLanguages || [],
  };
}

async function saveFixtureApplication(fixture) {
  const response = await fetch(`/api/applications/${encodeURIComponent(fixture.id)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(fixtureApplicationPayload(fixture)),
  });
  if (!response.ok) throw new Error((await response.text()).trim() || `Could not save ${fixture.id}`);
}

function fixtureByID(id) {
  return fixtures.find((fixture) => fixture.id === id);
}

async function loadFixtureImage(fixture) {
  const response = await fetch(`/static/fixtures/${encodeURIComponent(fixture.image)}`);
  if (!response.ok) throw new Error(`Could not load image ${fixture.image}`);
  const blob = await response.blob();
  return new File([blob], fixture.image, { type: blob.type || mimeTypeForImage(fixture.image) });
}

function assignFileToInput(input, file) {
  const transfer = new DataTransfer();
  transfer.items.add(file);
  input.files = transfer.files;
}

function updateSingleImageUI(file) {
  fileName.textContent = file ? file.name : "Choose label image";
  derivedApplicationId.textContent = file ? applicationIDFromFile(file) : "Choose an image to generate ID";
  previewFrame.classList.toggle("has-image", Boolean(file));
  if (!file) {
    imagePreview.style.display = "none";
    imagePreview.removeAttribute("src");
    return;
  }
  imagePreview.src = URL.createObjectURL(file);
  imagePreview.style.display = "block";
}

function setBatchRowFile(row, file) {
  const fileInput = row.querySelector(".batch-file-input");
  assignFileToInput(fileInput, file);
  row.querySelector(".batch-file-label span").textContent = file.name;
  row.querySelector(".batch-derived-id").textContent = applicationIDFromFile(file);
}

function mimeTypeForImage(name) {
  const lower = String(name).toLowerCase();
  if (lower.endsWith(".jpg") || lower.endsWith(".jpeg")) return "image/jpeg";
  if (lower.endsWith(".webp")) return "image/webp";
  return "image/png";
}

function applicationIDFromFile(file) {
  return dashedIDFromFilename(file?.name || "");
}

function dashedIDFromFilename(name) {
  const withoutExt = String(name).replace(/\.[^.]+$/, "");
  const id = withoutExt
    .normalize("NFKD")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
  return id || "label";
}

manageAppsButton.addEventListener("click", openManageModal);
manageModalClose.addEventListener("click", closeManageModal);
manageSelectAll.addEventListener("change", toggleSelectAllApplications);
manageDeleteSelected.addEventListener("click", deleteSelectedApplications);
manageModal.addEventListener("click", (event) => {
  if (event.target === manageModal) closeManageModal();
});
document.addEventListener("keydown", (event) => {
  if (event.key === "Escape" && !manageModal.hidden) closeManageModal();
});

async function openManageModal() {
  manageModal.hidden = false;
  await loadManageList();
}

function closeManageModal() {
  manageModal.hidden = true;
}

async function loadManageList() {
  manageList.innerHTML = `<li class="manage-empty">Loading…</li>`;
  updateManageSelectionState();
  let applications = [];
  try {
    const response = await fetch("/api/applications");
    if (!response.ok) throw new Error(await response.text());
    const payload = await response.json();
    applications = payload.applications || [];
  } catch (error) {
    manageList.innerHTML = `<li class="manage-empty">Could not load applications</li>`;
    updateManageSelectionState();
    return;
  }

  if (!applications.length) {
    manageList.innerHTML = `<li class="manage-empty">No applications yet</li>`;
    updateManageSelectionState();
    return;
  }

  manageList.innerHTML = applications
    .map((app) => {
      const id = app.ID || "";
      const meta = [app.Brand, app.ClassType].map((part) => String(part || "").trim()).filter(Boolean).join(" · ");
      return `
        <li class="manage-item" data-id="${escapeHtml(id)}">
          <div class="manage-row">
            <label class="manage-check" aria-label="Select ${escapeHtml(id)}">
              <input class="manage-select" type="checkbox" value="${escapeHtml(id)}" />
            </label>
            <div class="manage-row-main">
              <span class="manage-row-id">${escapeHtml(id)}</span>
              <span class="manage-row-meta">${escapeHtml(meta) || "<em>no metadata</em>"}</span>
            </div>
            <div class="manage-actions">
              <button type="button" class="small-button secondary manage-view">View</button>
              <button type="button" class="small-button danger manage-delete">Delete</button>
            </div>
          </div>
        </li>`;
    })
    .join("");

  manageList.querySelectorAll(".manage-item").forEach((item) => {
    const id = item.dataset.id;
    item.querySelector(".manage-select").addEventListener("change", updateManageSelectionState);
    item.querySelector(".manage-view").addEventListener("click", () => toggleManageDetail(item, id));
    item.querySelector(".manage-delete").addEventListener("click", () => deleteApplication(id));
  });
  updateManageSelectionState();
}

async function toggleManageDetail(item, id) {
  const existing = item.querySelector(".manage-detail");
  if (existing) {
    existing.remove();
    return;
  }

  const detail = document.createElement("div");
  detail.className = "manage-detail";
  detail.innerHTML = `<div class="manage-empty">Loading…</div>`;
  item.appendChild(detail);

  try {
    const response = await fetch(`/api/applications/${encodeURIComponent(id)}`);
    if (!response.ok) throw new Error(await response.text());
    const app = await response.json();
    detail.innerHTML = manageDetailFields(app)
      .filter((field) => field.value)
      .map(
        (field) => `
          <div class="manage-detail-field">
            <span>${escapeHtml(field.label)}</span>
            <p>${escapeHtml(field.value)}</p>
          </div>`,
      )
      .join("");
  } catch (error) {
    detail.innerHTML = `<div class="manage-empty">Could not load details</div>`;
  }
}

function manageDetailFields(app) {
  const foreignBlocks = (app.ForeignBlocks || [])
    .map((block) => [block.Text, block.EnglishTranslation].filter(Boolean).join(" => "))
    .filter(Boolean)
    .join("\n");
  return [
    { label: "Application ID", value: app.ID || "" },
    { label: "Brand", value: app.Brand || "" },
    { label: "Class / Type", value: app.ClassType || "" },
    { label: "Net Contents", value: app.NetContents || "" },
    { label: "Alcohol Content", value: app.ABV || "" },
    { label: "Name and Address", value: app.NameAndAddress || "" },
    { label: "Government Warning", value: app.GovernmentWarning || "" },
    { label: "Foreign Blocks", value: foreignBlocks },
  ];
}

async function deleteApplication(id) {
  const confirmed = window.confirm(
    `Delete application "${id}"?\n\nThis also removes its verification results and stored images. This cannot be undone.`,
  );
  if (!confirmed) return;

  try {
    await deleteApplicationRequest(id);
    setStatus(`Deleted application ${id}`);
  } catch (error) {
    setStatus(error.message || "Delete failed");
    return;
  }

  await loadManageList();
}

function updateManageSelectionState() {
  const boxes = Array.from(manageList.querySelectorAll(".manage-select"));
  const selected = boxes.filter((box) => box.checked).length;

  manageSelectAll.checked = boxes.length > 0 && selected === boxes.length;
  manageSelectAll.indeterminate = selected > 0 && selected < boxes.length;
  manageSelectAll.disabled = boxes.length === 0;
  manageDeleteSelected.disabled = selected === 0;
  manageSelectionCount.textContent = `${selected} selected`;
}

function toggleSelectAllApplications() {
  manageList.querySelectorAll(".manage-select").forEach((box) => {
    box.checked = manageSelectAll.checked;
  });
  updateManageSelectionState();
}

function selectedApplicationIDs() {
  return Array.from(manageList.querySelectorAll(".manage-select:checked"))
    .map((box) => box.value)
    .filter(Boolean);
}

async function deleteSelectedApplications() {
  const ids = selectedApplicationIDs();
  if (!ids.length) return;

  const confirmed = window.confirm(
    `Delete ${ids.length} selected application${ids.length === 1 ? "" : "s"}?\n\nThis also removes verification results and stored images. This cannot be undone.`,
  );
  if (!confirmed) return;

  manageDeleteSelected.disabled = true;
  const failed = [];
  for (const id of ids) {
    try {
      await deleteApplicationRequest(id);
    } catch (error) {
      failed.push(`${id}: ${error.message || "delete failed"}`);
    }
  }

  if (failed.length) {
    setStatus(`Deleted ${ids.length - failed.length} of ${ids.length}. Failed: ${failed.join("; ")}`);
  } else {
    setStatus(`Deleted ${ids.length} application${ids.length === 1 ? "" : "s"}`);
  }
  await loadManageList();
}

async function deleteApplicationRequest(id) {
  const response = await fetch(`/api/applications/${encodeURIComponent(id)}`, { method: "DELETE" });
  if (!response.ok) throw new Error((await response.text()).trim() || "Delete failed");
}

function renderVerdict(verdict, elapsed, run) {
  currentFields = verdict.Fields || [];
  selectedFieldIndex = -1;
  verdictBadge.textContent = verdict.Status || "unknown";
  verdictBadge.className = `verdict-badge ${verdict.Status || "idle"}`;
  confidenceText.textContent = `Confidence: ${formatScore(verdict.Confidence)} | ${elapsed} ms`;
  if (singleRunStamp && run) {
    singleRunStamp.textContent = `Run ${run.number} completed at ${run.time}`;
  }
  rawJson.textContent = JSON.stringify(verdict, null, 2);
  renderDecisionSummary(verdict, elapsed, currentFields);
  renderFailedFieldsPanel(currentFields);

  summaryStrip.innerHTML = currentFields
    .map((field) => `<span class="chip ${field.Pass ? "pass" : "fail"}">${escapeHtml(field.Field)}: ${field.Pass ? "pass" : "flag"}</span>`)
    .join("");

  if (!currentFields.length) {
    fieldsBody.innerHTML = `<tr><td colspan="5" class="empty">No fields returned</td></tr>`;
    renderFailedFieldsPanel([]);
    clearDetail();
    return;
  }

  fieldsBody.innerHTML = currentFields
    .map((field, index) => {
      const action = actionLabel(field);
      return `
        <tr class="field-row ${field.Pass ? "" : "flagged-row"}" data-index="${index}" tabindex="0">
          <td>
            <div class="field-name">${escapeHtml(field.Field)}</div>
            <div class="${field.Pass ? "pass-text" : "fail-text"}">${field.Pass ? "Pass" : "Flag"}</div>
          </td>
          <td>${evidenceCell(field)}</td>
          <td>${deterministicCell(field)}</td>
          <td>${judgeCell(field)}</td>
          <td><span class="action-pill ${field.Pass ? "pass" : "fail"}">${escapeHtml(action)}</span></td>
        </tr>`;
    })
    .join("");

  fieldsBody.querySelectorAll(".field-row").forEach((row) => {
    row.addEventListener("click", () => selectField(Number(row.dataset.index)));
    row.addEventListener("keydown", (event) => {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        selectField(Number(row.dataset.index));
      }
    });
  });

  const firstFailed = currentFields.findIndex((field) => !field.Pass);
  selectField(firstFailed >= 0 ? firstFailed : 0);
}

function resetVerificationOutput() {
  currentFields = [];
  selectedFieldIndex = -1;
  verdictBadge.textContent = "No run";
  verdictBadge.className = "verdict-badge idle";
  confidenceText.textContent = "Confidence: --";
  if (singleRunStamp) singleRunStamp.textContent = "No verification run yet";
  decisionStatus.textContent = "No run";
  decisionStatus.className = "";
  decisionRuntime.textContent = "--";
  decisionFailed.textContent = "--";
  decisionRisk.textContent = "--";
  decisionJudge.textContent = document.querySelector("#judgeEnabled").checked ? "Not needed" : "Off";
  summaryStrip.innerHTML = "";
  fieldsBody.innerHTML = `<tr><td colspan="5" class="empty">No verification run yet</td></tr>`;
  rawJson.textContent = "";
  renderFailedFieldsPanel([]);
  clearDetail();
  updateJudgeHint();
}

function renderFailedFieldsPanel(fields) {
  if (!failedFieldsPanel || !failedFieldsCount || !failedFieldsList) return;

  const failed = fields
    .map((field, index) => ({ field, index }))
    .filter((item) => !item.field.Pass);

  failedFieldsPanel.hidden = failed.length === 0;
  failedFieldsCount.textContent = `${failed.length} failed`;
  failedFieldsList.innerHTML = failed
    .map(
      ({ field, index }) => `
        <button type="button" class="failed-field-card" data-index="${index}">
          <span>${escapeHtml(field.Field)}</span>
          <strong>${escapeHtml(actionLabel(field))}</strong>
          <em>${escapeHtml(shortText(field.Diff || field.Extracted || "Missing text"))}</em>
        </button>`,
    )
    .join("");

  failedFieldsList.querySelectorAll(".failed-field-card").forEach((button) => {
    button.addEventListener("click", () => selectField(Number(button.dataset.index)));
  });
}

function renderDecisionSummary(verdict, elapsed, fields) {
  const failed = fields.filter((field) => !field.Pass);
  const reviewed = judgeReviewedFields(fields);
  const errors = judgeErrorFields(fields);
  const eligible = judgeEligibleFields(fields);
  const risk = failed.length
    ? failed.slice().sort((a, b) => (a.Score || 0) - (b.Score || 0))[0]
    : null;

  decisionStatus.textContent = verdict.Status || "unknown";
  decisionStatus.className = verdict.Status || "";
  decisionRuntime.textContent = `${elapsed} ms`;
  decisionFailed.textContent = String(failed.length);
  decisionRisk.textContent = risk ? `${risk.Field}: ${actionLabel(risk)}` : "None";
  decisionJudge.textContent = judgeDecisionText(reviewed.length, eligible.length, errors.length, failed.length);
  setJudgeRunStatus(judgeRunText(reviewed.length, eligible.length, errors.length, failed.length));
}

function completionMessage(verdict, elapsed) {
  if (!document.querySelector("#judgeEnabled").checked) return `Completed in ${elapsed} ms`;
  const fields = verdict.Fields || [];
  const failed = fields.filter((field) => !field.Pass).length;
  const reviewed = judgeReviewedFields(fields).length;
  const errors = judgeErrorFields(fields).length;
  const eligible = judgeEligibleFields(fields).length;
  if (reviewed > 0) return `Completed in ${elapsed} ms · AI Reader ${reviewPastTense(value("judgeMode"))} ${reviewed}`;
  if (errors > 0) return `Completed in ${elapsed} ms · AI Reader error`;
  if (eligible === 0 && failed === 0) return `Completed in ${elapsed} ms · AI Reader not needed`;
  if (eligible === 0) return `Completed in ${elapsed} ms · AI Reader no selected failures`;
  return `Completed in ${elapsed} ms · AI Reader no result`;
}

function judgeDecisionText(reviewed, eligible, errors = 0, failed = 0) {
  if (!document.querySelector("#judgeEnabled").checked) return "Off";
  if (reviewed > 0) return `${modeLabel(value("judgeMode"))} (${reviewed})`;
  if (errors > 0) return `Error (${errors})`;
  if (eligible === 0 && failed === 0) return "Not needed";
  if (eligible === 0) return "No selected failures";
  return `No result (${eligible} eligible)`;
}

function judgeRunText(reviewed, eligible, errors = 0, failed = 0) {
  if (!document.querySelector("#judgeEnabled").checked) {
    return "AI Label Reader is off. Deterministic OCR and matching will still run.";
  }
  if (reviewed > 0) {
    return `AI Label Reader returned evidence for ${reviewed} selected failed field${reviewed === 1 ? "" : "s"} in ${modeLabel(value("judgeMode"))} mode.`;
  }
  if (errors > 0) {
    return `AI Label Reader could not complete ${errors} read${errors === 1 ? "" : "s"}. The deterministic result was preserved.`;
  }
  if (eligible === 0 && failed === 0) {
    return "AI Label Reader was not needed because deterministic OCR and matching passed every field.";
  }
  if (eligible === 0) {
    return `AI Label Reader was not used because the failed fields were not selected for ${reviewNoun(value("judgeMode"))}. Selected fields: ${selectedJudgeFieldLabels("single")}.`;
  }
  return `AI Label Reader was requested for ${eligible} eligible field${eligible === 1 ? "" : "s"}, but no reader metadata came back. Check the judge service or timeout.`;
}

function judgeReviewedFields(fields) {
  return fields.filter((field) => ["llm", "llm_shadow", "ai_reader", "ai_reader_shadow"].includes(field.ReviewSource));
}

function judgeErrorFields(fields) {
  return fields.filter((field) => field.ReviewSource === "llm_error" || field.ReviewSource === "ai_reader_error");
}

function judgeEligibleFields(fields) {
  const allowed = new Set(selectedJudgeFields("single"));
  return fields.filter((field) => {
    const name = String(field.Field || "");
    return (
      !field.Pass &&
      allowed.has(name)
    );
  });
}

function evidenceCell(field) {
  return `
    <div class="evidence-block">
      <span>Expected</span>
      <strong>${escapeHtml(shortText(field.Expected || ""))}</strong>
    </div>
    <div class="evidence-block">
      <span>Deterministic OCR</span>
      <strong>${escapeHtml(shortText(field.Extracted || "")) || "<em>empty</em>"}</strong>
    </div>
    ${field.ReviewExtracted ? `
      <div class="evidence-block ai">
        <span>AI Reader</span>
        <strong>${escapeHtml(shortText(field.ReviewExtracted))}</strong>
      </div>` : ""}`;
}

function deterministicCell(field) {
  return `
    <div class="signal-stack">
      <span class="signal-label">${escapeHtml(field.MatchType || "match")}</span>
      <strong>${formatScore(deterministicScore(field))}</strong>
      <span>${field.Pass ? "accepted" : "failed"}</span>
    </div>`;
}

function judgeCell(field) {
  if (!field.ReviewSource) {
    return `<div class="signal-stack muted"><span>Not reviewed</span></div>`;
  }
  if (field.ReviewSource === "llm_error") {
    return `
      <div class="signal-stack muted">
        <span class="signal-label">AI Reader</span>
        <strong>Error</strong>
        <span>${escapeHtml(shortText(field.ReviewReason || "model unavailable"))}</span>
      </div>`;
  }
  if (field.ReviewSource === "ai_reader_error") {
    return `
      <div class="signal-stack muted">
        <span class="signal-label">AI Reader</span>
        <strong>Error</strong>
        <span>${escapeHtml(shortText(field.ReviewReason || "model unavailable"))}</span>
      </div>`;
  }
  return `
    <div class="signal-stack judge">
      <span class="signal-label">${escapeHtml(reviewSourceLabel(field.ReviewSource))}</span>
      <strong>${escapeHtml(field.ReviewDecision || "uncertain")}</strong>
      <span>${field.ReviewAccepted ? "accepted" : "not accepted"}</span>
    </div>`;
}

function selectField(index) {
  if (index < 0 || index >= currentFields.length) {
    clearDetail();
    return;
  }
  selectedFieldIndex = index;
  fieldsBody.querySelectorAll(".field-row").forEach((row) => {
    row.classList.toggle("selected", Number(row.dataset.index) === index);
  });
  renderFieldDetail(currentFields[index]);
}

function renderFieldDetail(field) {
  fieldDetail.classList.remove("empty-detail");
  fieldDetail.querySelector(".detail-head span").textContent = field.Pass ? "Verified" : actionLabel(field);
  detailBody.innerHTML = `
    <div class="detail-grid">
      <div>
        <span>Field</span>
        <strong>${escapeHtml(field.Field)}</strong>
      </div>
      <div>
        <span>Result</span>
        <strong class="${field.Pass ? "pass-text" : "fail-text"}">${field.Pass ? "Pass" : "Flag"}</strong>
      </div>
      <div>
        <span>Deterministic</span>
        <strong>${escapeHtml(field.MatchType || "match")} · ${formatScore(deterministicScore(field))}</strong>
      </div>
      <div>
        <span>AI Reader</span>
        <strong>${escapeHtml(judgeSummary(field))}</strong>
      </div>
    </div>
    <div class="compare-grid">
      <section>
        <h3>Expected</h3>
        <p>${escapeHtml(field.Expected || "")}</p>
      </section>
      <section>
        <h3>Deterministic Extracted</h3>
        <p>${escapeHtml(field.Extracted || "") || "<em>empty</em>"}</p>
      </section>
      ${field.ReviewExtracted ? `
        <section>
          <h3>AI Reader Extracted</h3>
          <p>${escapeHtml(field.ReviewExtracted)}</p>
        </section>` : ""}
    </div>
    <div class="detail-note">
      <span>Finding</span>
      <p>${escapeHtml(field.Pass ? "No discrepancy found for this field." : field.Diff || actionLabel(field))}</p>
    </div>
    ${field.ReviewReason ? `<div class="detail-note"><span>Reader Note</span><p>${escapeHtml(field.ReviewReason)}</p></div>` : ""}`;
}

function clearDetail() {
  fieldDetail.classList.add("empty-detail");
  fieldDetail.querySelector(".detail-head span").textContent = "Select a field row";
  detailBody.innerHTML = "";
}

function actionLabel(field) {
  if (field.Pass) return "Verified";
  const diff = String(field.Diff || "").toLowerCase();
  if (!String(field.Extracted || "").trim()) return "Missing text";
  if (field.MatchType === "format") return "Format mismatch";
  if (field.MatchType === "warning" || diff.includes("government warning")) return "Header mismatch";
  if (field.ReviewDecision === "not_equivalent") return "Reader disagreed";
  if (field.MatchType === "fuzzy" && typeof field.Score === "number" && field.Score < 0.6) return "Low similarity";
  if (diff.includes("expected")) return "Text mismatch";
  return "Needs review";
}

function judgeSummary(field) {
  if (!field.ReviewSource) return "Not reviewed";
  if (field.ReviewSource === "llm_error" || field.ReviewSource === "ai_reader_error") return "Error";
  return `${field.ReviewDecision || "uncertain"} · ${field.ReviewAccepted ? "accepted" : "not accepted"}`;
}

function deterministicScore(field) {
  return typeof field.DeterministicScore === "number" && field.DeterministicScore > 0
    ? field.DeterministicScore
    : field.Score;
}

function modeLabel(mode) {
  return mode === "override" ? "reconcile" : "compare";
}

function reviewPastTense(mode) {
  return mode === "override" ? "reconciled" : "compared";
}

function reviewNoun(mode) {
  return mode === "override" ? "reconciliation" : "comparison";
}

function reviewSourceLabel(source) {
  switch (source) {
    case "ai_reader":
      return "AI reader";
    case "ai_reader_shadow":
      return "AI reader compare";
    case "llm":
      return "Text judge";
    case "llm_shadow":
      return "Text judge compare";
    default:
      return source || "AI reader";
  }
}

function shortText(text) {
  const value = String(text || "").trim();
  if (value.length <= 96) return value;
  return `${value.slice(0, 93)}...`;
}

function value(id) {
  return document.querySelector(`#${id}`)?.value.trim() || "";
}

function setValue(id, nextValue) {
  const element = document.querySelector(`#${id}`);
  if (element) element.value = nextValue || "";
}

function selectedJudgeFields(scope) {
  return Array.from(document.querySelectorAll(`[data-judge-field="${scope}"]:checked`)).map((input) => input.value);
}

function selectedJudgeFieldLabels(scope) {
  const labels = {
    brand: "brand",
    class_type: "class/type",
    net_contents: "net contents",
    abv: "alcohol content",
    government_warning: "government warning",
    name_address: "name/address",
  };
  const selected = selectedJudgeFields(scope).map((field) => labels[field] || field);
  return selected.length ? selected.join(", ") : "no fields";
}

function updateJudgeHint() {
  if (!document.querySelector("#judgeEnabled").checked) {
    setJudgeRunStatus("AI Label Reader is off. Deterministic OCR and matching will still run.");
    return;
  }
  setJudgeRunStatus(
    `AI Label Reader runs in parallel in ${modeLabel(value("judgeMode"))} mode. Selected failed fields: ${selectedJudgeFieldLabels("single")}.`,
  );
}

function setJudgeRunStatus(message) {
  judgeRunStatus.textContent = message;
}

function setApplicationSaveStatus(message, state) {
  applicationSaveStatus.textContent = message;
  applicationSaveStatus.className = `inline-status ${state || ""}`.trim();
}

function setBusy(isBusy, message) {
  verifyButton.disabled = isBusy;
  form.querySelectorAll("button").forEach((button) => {
    button.disabled = isBusy;
  });
  setStatus(message);
}

function setStatus(message) {
  statusText.textContent = message;
  return new Error(message);
}

function formatScore(score) {
  if (typeof score !== "number" || Number.isNaN(score)) return "--";
  return score.toFixed(3).replace(/0+$/, "").replace(/\.$/, "");
}

function formatDuration(ms) {
  if (typeof ms !== "number" || Number.isNaN(ms)) return "--";
  if (ms < 1000) return `${Math.round(ms)} ms`;
  return `${(ms / 1000).toFixed(1)} s`;
}

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

window.addEventListener("error", (event) => {
  setBusy(false, event.error?.message || "Unexpected error");
  setBatchBusy(false, event.error?.message || "Unexpected error");
});

window.addEventListener("unhandledrejection", (event) => {
  setBusy(false, event.reason?.message || "Unexpected error");
  setBatchBusy(false, event.reason?.message || "Unexpected error");
});
