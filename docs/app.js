const fileInput = document.getElementById("file-input");
const dropZone = document.getElementById("drop-zone");
const settingsForm = document.getElementById("settings-form");
const statusList = document.getElementById("status-list");
const statusEmpty = document.getElementById("status-empty");
const resetBtn = document.getElementById("reset-btn");

const state = {
  files: [],
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

function resetStatus() {
  statusList.innerHTML = "";
  statusList.hidden = true;
  statusEmpty.hidden = false;
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

  for (const file of state.files) {
    const result = await compressFile(file, settings);
    renderResult(result, file.name);
  }
}

function registerFiles(fileList) {
  const files = Array.from(fileList).filter((file) => file.type === "image/jpeg");
  if (!files.length) {
    alert("Похоже, JPEG файлов нет.");
    return;
  }
  state.files = files;
}

fileInput.addEventListener("change", (event) => {
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
});

dropZone.addEventListener("dragover", (event) => {
  event.preventDefault();
  dropZone.classList.add("dragover");
});

dropZone.addEventListener("dragleave", () => {
  dropZone.classList.remove("dragover");
});

dropZone.addEventListener("drop", (event) => {
  event.preventDefault();
  dropZone.classList.remove("dragover");
  registerFiles(event.dataTransfer.files);
});
