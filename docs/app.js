const fileInput = document.getElementById("file-input");
const fileZone = document.getElementById("file-zone");
const folderZone = document.getElementById("folder-zone");
const folderInput = document.getElementById("folder-input");
const dropZone = document.getElementById("drop-zone");
const settingsForm = document.getElementById("settings-form");
const statusList = document.getElementById("status-list");
const statusEmpty = document.getElementById("status-empty");
const resetBtn = document.getElementById("reset-btn");
const downloadAllBtn = document.getElementById("download-all");
const fileList = document.getElementById("file-list");

const state = {
  files: [],
  results: [],
  processed: 0,
};

const DEFAULTS = {
  targetKB: 300,
  initialQuality: 85,
  minQuality: 55,
  qualityStep: 5,
  maxWidth: 2380,
  maxHeight: 1600,
  minWidth: 1290,
  minHeight: 800,
};

function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value));
}

function getSettings() {
  const targetKB = parseInt(document.getElementById("target-kb").value, 10);
  const initialQuality = parseInt(document.getElementById("initial-quality").value, 10);
  const minQuality = parseInt(document.getElementById("min-quality").value, 10);
  const qualityStep = parseInt(document.getElementById("quality-step").value, 10);
  const maxWidth = parseInt(document.getElementById("max-width").value, 10);
  const maxHeight = parseInt(document.getElementById("max-height").value, 10);
  const minWidth = parseInt(document.getElementById("min-width").value, 10);
  const minHeight = parseInt(document.getElementById("min-height").value, 10);

  return {
    targetBytes: clamp(targetKB, 10, 5000) * 1024,
    initialQuality: clamp(initialQuality, 1, 100),
    minQuality: clamp(minQuality, 1, 100),
    qualityStep: clamp(qualityStep, 1, 20),
    maxWidth: clamp(maxWidth, 0, 10000),
    maxHeight: clamp(maxHeight, 0, 10000),
    minWidth: clamp(minWidth, 0, 10000),
    minHeight: clamp(minHeight, 0, 10000),
  };
}

function determineScaleFactor(width, height, bounds) {
  if (width <= 0 || height <= 0) {
    return 1;
  }

  let lower = 0;
  let upper = Infinity;

  if (bounds.minWidth > 0) {
    lower = Math.max(lower, bounds.minWidth / width);
  }
  if (bounds.minHeight > 0) {
    lower = Math.max(lower, bounds.minHeight / height);
  }
  if (bounds.maxWidth > 0) {
    upper = Math.min(upper, bounds.maxWidth / width);
  }
  if (bounds.maxHeight > 0) {
    upper = Math.min(upper, bounds.maxHeight / height);
  }

  if (lower <= 1 && 1 <= upper) {
    return 1;
  }
  if (lower > upper) {
    return upper < 1 ? upper : lower;
  }
  if (lower > 1) {
    return lower;
  }
  if (upper < 1) {
    return upper;
  }
  return 1;
}

function formatSize(bytes) {
  return `${(bytes / 1024).toFixed(1)} KB`;
}

function setStatusBusy() {
  statusList.hidden = false;
  statusEmpty.hidden = true;
}

function updateDownloadAll() {
  const remaining = Math.max(state.files.length - state.processed, 0);
  downloadAllBtn.disabled = remaining !== 0 || state.results.length === 0;
  downloadAllBtn.textContent = `Скачать все (${remaining} осталось)`;
}

function resetStatus() {
  statusList.innerHTML = "";
  statusList.hidden = true;
  statusEmpty.hidden = false;
  state.results = [];
  state.processed = 0;
  downloadAllBtn.hidden = true;
  updateDownloadAll();
}

async function fileToImageBitmap(file) {
  const buffer = await file.arrayBuffer();
  const blob = new Blob([buffer]);
  return createImageBitmap(blob);
}

async function encodeCanvas(canvas, quality) {
  return new Promise((resolve) => {
    canvas.toBlob(
      (blob) => resolve(blob),
      "image/jpeg",
      clamp(quality / 100, 0.01, 1)
    );
  });
}

async function compressFile(file, settings) {
  const imageBitmap = await fileToImageBitmap(file);
  const scale = determineScaleFactor(imageBitmap.width, imageBitmap.height, settings);
  const targetWidth = Math.max(1, Math.round(imageBitmap.width * scale));
  const targetHeight = Math.max(1, Math.round(imageBitmap.height * scale));

  const canvas = document.createElement("canvas");
  canvas.width = targetWidth;
  canvas.height = targetHeight;

  const ctx = canvas.getContext("2d", { alpha: false });
  ctx.imageSmoothingEnabled = true;
  ctx.imageSmoothingQuality = "high";
  ctx.drawImage(imageBitmap, 0, 0, targetWidth, targetHeight);

  let bestBlob = null;
  let bestQuality = settings.initialQuality;

  for (let quality = settings.initialQuality; quality >= settings.minQuality; quality -= settings.qualityStep) {
    const blob = await encodeCanvas(canvas, quality);
    if (!blob) {
      break;
    }
    if (blob.size <= settings.targetBytes) {
      return {
        blob,
        quality,
        original: `${imageBitmap.width}x${imageBitmap.height}`,
        processed: `${targetWidth}x${targetHeight}`,
        label: "OK",
      };
    }
    if (!bestBlob || blob.size < bestBlob.size) {
      bestBlob = blob;
      bestQuality = quality;
    }
  }

  return {
    blob: bestBlob,
    quality: bestQuality,
    original: `${imageBitmap.width}x${imageBitmap.height}`,
    processed: `${targetWidth}x${targetHeight}`,
    label: "MAXED",
  };
}

function renderResult(result, originalName) {
  if (!result.blob) {
    return;
  }

  const item = document.createElement("div");
  item.className = "status-item";

  const meta = document.createElement("div");
  meta.className = "status-meta";

  const title = document.createElement("strong");
  title.textContent = `${originalName} — ${result.label}`;

  const size = document.createElement("span");
  size.textContent = `Размер: ${formatSize(result.blob.size)} · q=${result.quality}`;

  const dims = document.createElement("span");
  dims.textContent = `Габариты: ${result.original} → ${result.processed}`;

  meta.append(title, size, dims);

  const link = document.createElement("a");
  link.className = "button";
  link.textContent = "Скачать";
  link.href = URL.createObjectURL(result.blob);
  const suffix = originalName.replace(/\.jpe?g$/i, "");
  link.download = `${suffix}_compressed.jpg`;

  item.append(meta, link);
  statusList.append(item);
  state.results.push({ name: link.download, blob: result.blob });
  state.processed += 1;
  updateDownloadAll();
}

async function handleCompress(event) {
  event.preventDefault();
  if (!state.files.length) {
    alert("Добавь хотя бы один JPEG.");
    return;
  }

  const settings = getSettings();
  if (settings.minQuality > settings.initialQuality) {
    alert("Минимальное качество не может быть больше стартового.");
    return;
  }

  resetStatus();
  setStatusBusy();
  downloadAllBtn.hidden = false;
  updateDownloadAll();

  for (const file of state.files) {
    const result = await compressFile(file, settings);
    renderResult(result, file.name);
  }
}

function renderFileList(files) {
  fileList.innerHTML = "";
  if (!files.length) {
    const empty = document.createElement("div");
    empty.className = "file-empty";
    empty.textContent = "Файлы еще не добавлены.";
    fileList.append(empty);
    return;
  }

  const folders = new Map();
  files.forEach((file) => {
    const path = file.webkitRelativePath || file._relativePath || file.name;
    const parts = path.split("/");
    const folder = parts.length > 1 ? parts[0] : "Файлы без папки";
    if (!folders.has(folder)) {
      folders.set(folder, []);
    }
    folders.get(folder).push(path);
  });

  for (const [folder, paths] of folders.entries()) {
    const folderRow = document.createElement("div");
    folderRow.className = "file-item file-folder";
    folderRow.textContent = folder;
    fileList.append(folderRow);

    paths.forEach((path) => {
      const row = document.createElement("div");
      row.className = "file-item";
      row.textContent = `• ${path}`;
      fileList.append(row);
    });
  }
}

function registerFiles(fileListInput) {
  const files = Array.from(fileListInput).filter((file) => {
    if (file.type === "image/jpeg") {
      return true;
    }
    const name = (file.name || "").toLowerCase();
    return name.endsWith(".jpg") || name.endsWith(".jpeg");
  });
  if (!files.length) {
    alert("Похоже, JPEG файлов нет.");
    return;
  }
  state.files = files;
  renderFileList(files);
  updateDownloadAll();
}

fileZone.addEventListener("click", () => {
  fileInput.click();
});

folderZone.addEventListener("click", () => {
  folderInput.click();
});

fileZone.addEventListener("keydown", (event) => {
  if (event.key === "Enter" || event.key === " ") {
    event.preventDefault();
    fileInput.click();
  }
});

folderZone.addEventListener("keydown", (event) => {
  if (event.key === "Enter" || event.key === " ") {
    event.preventDefault();
    folderInput.click();
  }
});

fileInput.addEventListener("change", (event) => {
  registerFiles(event.target.files);
});

folderInput.addEventListener("change", (event) => {
  registerFiles(event.target.files);
});
settingsForm.addEventListener("submit", handleCompress);

resetBtn.addEventListener("click", () => {
  settingsForm.reset();
  document.getElementById("target-kb").value = DEFAULTS.targetKB;
  document.getElementById("initial-quality").value = DEFAULTS.initialQuality;
  document.getElementById("min-quality").value = DEFAULTS.minQuality;
  document.getElementById("quality-step").value = DEFAULTS.qualityStep;
  document.getElementById("max-width").value = DEFAULTS.maxWidth;
  document.getElementById("max-height").value = DEFAULTS.maxHeight;
  document.getElementById("min-width").value = DEFAULTS.minWidth;
  document.getElementById("min-height").value = DEFAULTS.minHeight;
  resetStatus();
  state.files = [];
  renderFileList([]);
});

async function collectDroppedFiles(items) {
  const files = [];

  async function walkEntry(entry, prefix) {
    if (entry.isFile) {
      const file = await new Promise((resolve) => entry.file(resolve));
      file._relativePath = `${prefix}${file.name}`;
      files.push(file);
      return;
    }
    if (entry.isDirectory) {
      const reader = entry.createReader();
      const dirPrefix = `${prefix}${entry.name}/`;
      while (true) {
        const entries = await new Promise((resolve) => reader.readEntries(resolve));
        if (!entries.length) {
          break;
        }
        await Promise.all(entries.map((child) => walkEntry(child, dirPrefix)));
      }
    }
  }

  const entries = Array.from(items)
    .map((item) => (item.webkitGetAsEntry ? item.webkitGetAsEntry() : null))
    .filter(Boolean);

  await Promise.all(entries.map((entry) => walkEntry(entry, "")));
  return files;
}

fileZone.addEventListener("dragover", (event) => {
  event.preventDefault();
  event.stopPropagation();
  dropZone.classList.add("dragover");
});

folderZone.addEventListener("dragover", (event) => {
  event.preventDefault();
  event.stopPropagation();
  dropZone.classList.add("dragover");
});

fileZone.addEventListener("dragleave", () => {
  dropZone.classList.remove("dragover");
});

folderZone.addEventListener("dragleave", () => {
  dropZone.classList.remove("dragover");
});

fileZone.addEventListener("drop", async (event) => {
  event.preventDefault();
  event.stopPropagation();
  dropZone.classList.remove("dragover");
  if (event.dataTransfer.items && event.dataTransfer.items.length) {
    const files = await collectDroppedFiles(event.dataTransfer.items);
    registerFiles(files);
  } else {
    registerFiles(event.dataTransfer.files);
  }
});

folderZone.addEventListener("drop", async (event) => {
  event.preventDefault();
  event.stopPropagation();
  dropZone.classList.remove("dragover");
  if (event.dataTransfer.items && event.dataTransfer.items.length) {
    const files = await collectDroppedFiles(event.dataTransfer.items);
    registerFiles(files);
  } else {
    registerFiles(event.dataTransfer.files);
  }
});

downloadAllBtn.addEventListener("click", async () => {
  if (downloadAllBtn.disabled || !state.results.length) {
    return;
  }
  const zip = new JSZip();
  state.results.forEach((result) => {
    zip.file(result.name, result.blob);
  });
  const blob = await zip.generateAsync({ type: "blob" });
  const link = document.createElement("a");
  link.href = URL.createObjectURL(blob);
  link.download = "jpgtools_compressed.zip";
  link.click();
  URL.revokeObjectURL(link.href);
});
