// Экран регистрации
const btnBack = document.getElementById("btn-back");
const btnRegister = document.getElementById("btn-register");
const usernameInput = document.getElementById("input-username");
const btnMic = document.getElementById("btn-mic");
const btnCamera = document.getElementById("btn-camera");
const cameraPreview = document.getElementById("camera-preview");
const cameraPreviewVideo = document.getElementById("camera-preview-video");
const micVolumeContainer = document.getElementById("mic-volume-slider");
const micVolumeRange = document.getElementById("mic-volume-range");

const API_BASE = window.location.origin + "/api/v1";

let isMicEnabled = false;
let isCameraEnabled = false;
let cameraStream = null;
let micStream = null;
let audioContext = null;
let analyser = null;

// Сохранение токенов в localStorage
function saveTokens(accessToken, refreshToken) {
  localStorage.setItem("accessToken", accessToken);
  localStorage.setItem("refreshToken", refreshToken);
}

function updateRegisterButtonState() {
  const hasUsername = usernameInput.value.trim().length > 0;
  btnRegister.disabled = !hasUsername;
  btnRegister.classList.toggle("enabled", hasUsername);
}

usernameInput.addEventListener("input", updateRegisterButtonState);
updateRegisterButtonState();

btnBack.addEventListener("click", () => {
  window.location.href = "MainMenu.html";
});

btnRegister.addEventListener("click", async () => {
  if (btnRegister.disabled) return;

  const username = usernameInput.value.trim();

  if (!username) {
    alert("Введите имя пользователя");
    return;
  }

  btnRegister.disabled = true;
  btnRegister.textContent = "Создание комнаты...";

  try {
    // Останавливаем потоки перед переходом
    if (cameraStream) {
      cameraStream.getTracks().forEach(track => track.stop());
    }
    if (micStream) {
      micStream.getTracks().forEach(track => track.stop());
    }
    if (audioContext) {
      audioContext.close();
    }

    // Сохранение имени пользователя и настроек медиа
    localStorage.setItem("display_name", username);
    localStorage.setItem("username", username); // Для совместимости
    localStorage.setItem("micEnabled", isMicEnabled);
    localStorage.setItem("cameraEnabled", isCameraEnabled);

    // Переход в комнату (комната будет создана автоматически при загрузке)
    window.location.href = "room.html";
  } catch (error) {
    console.error("Error:", error);
    alert("Ошибка при создании комнаты");
    btnRegister.disabled = false;
    btnRegister.textContent = "Создать комнату";
  }
});

async function toggleCamera() {
  if (!isCameraEnabled) {
    // Включаем камеру
    try {
      cameraStream = await navigator.mediaDevices.getUserMedia({ video: true });
      cameraPreviewVideo.srcObject = cameraStream;
      cameraPreview.classList.remove("hidden");
      isCameraEnabled = true;
      btnCamera.classList.add("active");
    } catch (error) {
      console.error("Ошибка доступа к камере:", error);
      alert("Не удалось получить доступ к камере");
    }
  } else {
    // Выключаем камеру
    if (cameraStream) {
      cameraStream.getTracks().forEach(track => track.stop());
      cameraStream = null;
    }
    cameraPreviewVideo.srcObject = null;
    cameraPreview.classList.add("hidden");
    isCameraEnabled = false;
    btnCamera.classList.remove("active");
  }
}

async function toggleMicrophone() {
  if (!isMicEnabled) {
    // Включаем микрофон
    try {
      micStream = await navigator.mediaDevices.getUserMedia({ audio: true });
      
      // Создаем аудио контекст для анализа громкости
      audioContext = new (window.AudioContext || window.webkitAudioContext)();
      analyser = audioContext.createAnalyser();
      const source = audioContext.createMediaStreamSource(micStream);
      source.connect(analyser);
      analyser.fftSize = 256;
      
      // Показываем ползунок громкости
      micVolumeContainer.classList.remove("hidden");
      isMicEnabled = true;
      btnMic.classList.add("active");
      
      // Обновляем ползунок на основе уровня громкости
      updateVolumeSlider();
    } catch (error) {
      console.error("Ошибка доступа к микрофону:", error);
      alert("Не удалось получить доступ к микрофону");
    }
  } else {
    // Выключаем микрофон
    if (micStream) {
      micStream.getTracks().forEach(track => track.stop());
      micStream = null;
    }
    if (audioContext) {
      audioContext.close();
      audioContext = null;
    }
    analyser = null;
    micVolumeContainer.classList.add("hidden");
    isMicEnabled = false;
    btnMic.classList.remove("active");
  }
}

function updateVolumeSlider() {
  if (!analyser || !isMicEnabled) return;
  
  const dataArray = new Uint8Array(analyser.frequencyBinCount);
  analyser.getByteFrequencyData(dataArray);
  
  // Вычисляем средний уровень громкости
  const average = dataArray.reduce((a, b) => a + b) / dataArray.length;
  const volume = Math.round((average / 255) * 100);
  
  // Обновляем значение ползунка визуально (но не меняем его значение вручную)
  // Пользователь может сам регулировать громкость через ползунок
  
  requestAnimationFrame(updateVolumeSlider);
}

btnMic.addEventListener("click", toggleMicrophone);
btnCamera.addEventListener("click", toggleCamera);

// Enter для отправки формы
usernameInput.addEventListener("keydown", (e) => {
  if (e.key === "Enter" && !btnRegister.disabled) {
    btnRegister.click();
  }
});

