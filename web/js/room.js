// Комната видеоконференции с LiveKit

// === РАННЯЯ ДИАГНОСТИКА (выполняется сразу при загрузке скрипта) ===
console.log("=== room.js загружен ===");
console.log("Текущий URL:", window.location.href);
console.log("URL search:", window.location.search);
console.log("URL searchParams room:", new URLSearchParams(window.location.search).get("room"));
console.log("localStorage.current_room_id:", localStorage.getItem("current_room_id"));
console.log("sessionStorage.current_room_id:", sessionStorage.getItem("current_room_id"));
console.log("=== Конец ранней диагностики ===");

const roomBanner = document.getElementById("banner-room");
const roomSubtitle = document.getElementById("room-subtitle");
const viewDefault = document.getElementById("room-view-default");
const viewChat = document.getElementById("room-view-chat");
const viewScreen = document.getElementById("room-view-screen");

const btnRoomMic = document.getElementById("room-mic");
const btnRoomCamera = document.getElementById("room-camera");
const btnRoomScreen = document.getElementById("room-screen");
const btnRoomChat = document.getElementById("room-chat");
const btnRoomLeave = document.getElementById("room-leave");
const btnRoomSettings = document.getElementById("room-settings");

// Элементы модального окна выбора устройств
const deviceSettingsModal = document.getElementById("device-settings-modal");
const cameraSelect = document.getElementById("camera-select");
const microphoneSelect = document.getElementById("microphone-select");
const speakerSelect = document.getElementById("speaker-select");
const deviceSettingsCancel = document.getElementById("device-settings-cancel");
const deviceSettingsApply = document.getElementById("device-settings-apply");

const chatMessages = document.getElementById("chat-messages");
const chatInput = document.getElementById("chat-input");
const chatSend = document.getElementById("chat-send");
const copyRoomIdBtn = document.getElementById("copy-room-id-btn");

// Элементы списка участников
const participantsModal = document.getElementById("participants-modal");
const participantsList = document.getElementById("participants-list");
const participantsCount = document.getElementById("participants-count");
const participantsClose = document.getElementById("participants-close");
const btnRoomParticipants = document.getElementById("room-participants");

// Элементы для видео
const localVideoElement = document.getElementById("local-video");
const remoteVideoElement = document.getElementById("remote-video");
const localVideoChatElement = document.getElementById("local-video-chat");
const remoteVideoChatElement = document.getElementById("remote-video-chat");
const localVideoScreenElement = document.getElementById("local-video-screen");
const remoteVideoScreenElement = document.getElementById("remote-video-screen");
const screenShareVideoElement = document.getElementById("screen-share-video");

const API_BASE = window.location.origin + "/api/v1";
let currentRoomId = null;

// Функция показа уведомлений
function showNotification(message, type = "info") {
  // Удаляем предыдущее уведомление если есть
  const existing = document.getElementById("notification-toast");
  if (existing) existing.remove();

  const toast = document.createElement("div");
  toast.id = "notification-toast";
  toast.style.cssText = `
    position: fixed;
    top: 20px;
    left: 50%;
    transform: translateX(-50%);
    padding: 12px 24px;
    border-radius: 8px;
    color: white;
    font-weight: 500;
    z-index: 10000;
    animation: fadeIn 0.3s ease;
    max-width: 80%;
    text-align: center;
  `;

  if (type === "warning") {
    toast.style.background = "#f59e0b";
  } else if (type === "error") {
    toast.style.background = "#ef4444";
  } else {
    toast.style.background = "#3b82f6";
  }

  toast.textContent = message;
  document.body.appendChild(toast);

  // Автоматически скрываем через 5 секунд
  setTimeout(() => {
    toast.style.opacity = "0";
    toast.style.transition = "opacity 0.3s";
    setTimeout(() => toast.remove(), 300);
  }, 5000);
}

// Кэш для информации о сервере
let serverInfo = null;

// Получение информации о сервере (IP, порт LiveKit)
async function getServerInfo() {
  if (serverInfo) {
    return serverInfo;
  }

  try {
    const response = await fetch(window.location.origin + "/server-info");
    if (response.ok) {
      serverInfo = await response.json();
      console.log("Получена информация о сервере:", serverInfo);
      return serverInfo;
    }
  } catch (error) {
    console.warn("Не удалось получить информацию о сервере:", error);
  }

  // Fallback - используем текущий хост
  const currentHost = window.location.hostname;
  serverInfo = {
    host_ip: currentHost === "localhost" ? "localhost" : currentHost,
    livekit_port: window.location.protocol === 'https:' ? '443' : '7880',
    livekit_url: (window.location.protocol === 'https:' ? 'wss://' + currentHost : 'ws://' + currentHost + ':7880'),
  };
  console.log("Используем fallback информацию о сервере:", serverInfo);
  return serverInfo;
}

// Преобразование LiveKit URL для работы в локальной сети
function transformLiveKitUrl(url) {
  if (!url) return url;

  const currentHost = window.location.hostname;
  let transformedUrl = url;

  // ВАЖНО: Всегда используем тот же хост, что и в URL браузера
  // Это избегает CORS проблем (origin должен совпадать)
  try {
    const urlObj = new URL(url.replace('ws://', 'http://').replace('wss://', 'https://'));
    const port = urlObj.port || '7880';

    // Если открыли через localhost - используем localhost
    // Если открыли через IP - используем тот же IP
    const wsProto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const portPart = window.location.protocol === 'https:' ? '' : ':' + port;
    transformedUrl = wsProto + '//' + currentHost + portPart;
  } catch (e) {
    // Fallback: просто заменяем хост
    transformedUrl = url
      .replace(/localhost/g, currentHost)
      .replace(/127\.0\.0\.1/g, currentHost)
      .replace(/192\.168\.\d+\.\d+/g, currentHost)
      .replace(/172\.\d+\.\d+\.\d+/g, currentHost)  // Docker internal IPs
      .replace(/10\.\d+\.\d+\.\d+/g, currentHost)   // Docker bridge network
      .replace(/livekit:/g, currentHost + ":");
  }

  // Убеждаемся, что протокол правильный
  if (!transformedUrl.startsWith("ws://") && !transformedUrl.startsWith("wss://")) {
    transformedUrl = "ws://" + transformedUrl;
  }

  console.log("LiveKit URL трансформирован:", url, "->", transformedUrl);
  return transformedUrl;
}

let room = null;
let localVideoTrack = null;
let localAudioTrack = null;
let screenTrack = null;
let isMicEnabled = false;
let isCameraEnabled = false;
let isScreenSharing = false;
let currentRoomView = "default";

// Server-side screen sharing WebRTC
let serverScreenSharePC = null;
let serverScreenSharePeerId = null;

// Выбранные устройства
let selectedCameraId = null;
let selectedMicrophoneId = null;
let selectedSpeakerId = null;

// Список доступных устройств
let availableDevices = {
  videoInputs: [],
  audioInputs: [],
  audioOutputs: []
};

// Хранилище для треков удаленных участников
const remoteParticipants = new Map(); // participant.identity -> { videoTrack, audioTrack, element, audioContext, analyser, volume }

// Активный говорящий/демонстрирующий
let activeSpeakerId = null;
let activeScreenShareId = null;

// Анализаторы аудио для определения активного говорящего
const audioAnalysers = new Map(); // participant.identity -> { audioContext, analyser, dataArray }

// Генерация UUID (с fallback для браузеров без crypto.randomUUID)
function generateUUID() {
  // Используем crypto.randomUUID если доступен (HTTPS или localhost)
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    try {
      return crypto.randomUUID();
    } catch (e) {
      // Fallback если randomUUID недоступен
    }
  }

  // Fallback: генерация UUID v4 вручную
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function (c) {
    const r = Math.random() * 16 | 0;
    const v = c === 'x' ? r : (r & 0x3 | 0x8);
    return v.toString(16);
  });
}

// Генерация и получение participant_id
function getParticipantID() {
  let participantID = localStorage.getItem("participant_id");
  if (!participantID) {
    participantID = generateUUID();
    localStorage.setItem("participant_id", participantID);
  }
  return participantID;
}

// Получение заголовков для API запросов
function getAPIHeaders() {
  const participantID = getParticipantID();
  const headers = {
    "X-Participant-ID": participantID,
    "Content-Type": "application/json",
  };
  console.log("API Headers:", headers);
  return headers;
}

// Функция для получения LiveKit SDK из различных источников
function getLiveKitSDK() {
  // Проверяем все возможные варианты глобальной переменной
  if (typeof window !== "undefined") {
    // UMD версия обычно экспортирует как window.LiveKitClient
    if (window.LiveKitClient) {
      if (window.LiveKitClient.Room) {
        return window.LiveKitClient;
      }
      // Проверяем, может быть это объект с другими свойствами
      if (window.LiveKitClient.default && window.LiveKitClient.default.Room) {
        return window.LiveKitClient.default;
      }
    }
    if (window.LiveKit && window.LiveKit.Room) {
      return window.LiveKit;
    }
    if (window.livekit && window.livekit.Room) {
      return window.livekit;
    }
    // Проверяем все свойства window, которые могут содержать LiveKit
    for (const key in window) {
      if (key.toLowerCase().includes('livekit')) {
        const obj = window[key];
        if (obj && typeof obj === 'object' && obj.Room) {
          console.log("Найден LiveKit SDK в window." + key);
          return obj;
        }
        if (obj && typeof obj === 'object' && obj.default && obj.default.Room) {
          console.log("Найден LiveKit SDK в window." + key + ".default");
          return obj.default;
        }
      }
    }
  }

  // Проверяем глобальные переменные без window (в строгом режиме это может не работать)
  try {
    if (typeof LiveKitClient !== "undefined") {
      if (LiveKitClient.Room) {
        return LiveKitClient;
      }
      if (LiveKitClient.default && LiveKitClient.default.Room) {
        return LiveKitClient.default;
      }
    }
    if (typeof LiveKit !== "undefined" && LiveKit.Room) {
      return LiveKit;
    }
  } catch (e) {
    // Игнорируем ошибки в строгом режиме
  }

  return null;
}

// Получение и валидация displayName
function getDisplayName() {
  const stored = localStorage.getItem("display_name");
  // Гарантируем, что displayName всегда имеет валидное значение
  let displayName = (stored && stored.trim()) || "User";

  // Ограничиваем длину (максимум 50 символов, как на бэкенде)
  const maxLength = 50;
  if (displayName.length > maxLength) {
    displayName = displayName.substring(0, maxLength);
  }

  // Сохраняем нормализованное значение обратно
  if (!stored || stored.trim() !== displayName) {
    localStorage.setItem("display_name", displayName);
  }

  return displayName;
}

// Присоединение к существующей комнате
async function joinExistingRoom(roomId, displayName) {
  const joinResponse = await fetch(`${API_BASE}/rooms/${roomId}/join`, {
    method: "POST",
    headers: getAPIHeaders(),
    body: JSON.stringify({ display_name: displayName }),
  });

  if (!joinResponse.ok) {
    const error = await joinResponse.json().catch(() => ({}));
    if (joinResponse.status === 404) {
      throw new Error("Комната не найдена. Проверьте правильность ID комнаты.");
    }
    throw new Error(error.error || "Не удалось присоединиться к комнате");
  }

  return await joinResponse.json();
}

// Создание новой комнаты
async function createNewRoom(displayName) {
  // Гарантируем, что displayName всегда валидное
  const validDisplayName = displayName && displayName.trim() ? displayName.trim() : "User";

  const requestBody = {
    title: "Новая комната",
    max_participants: 10,
    display_name: validDisplayName,
  };

  console.log("Creating room with data:", requestBody);

  const createResponse = await fetch(`${API_BASE}/rooms`, {
    method: "POST",
    headers: getAPIHeaders(),
    body: JSON.stringify(requestBody),
  });

  if (!createResponse.ok) {
    const error = await createResponse.json().catch(() => ({}));
    const errorMessage = error.error || "Не удалось создать комнату";
    console.error("Room creation failed:", errorMessage);
    throw new Error(errorMessage);
  }

  return await createResponse.json();
}

// Управление сохранённой комнатой в localStorage
const ROOM_STORAGE_KEY = "current_room_id";

function getSavedRoomId() {
  // Проверяем localStorage и sessionStorage
  const fromLocal = localStorage.getItem(ROOM_STORAGE_KEY);
  const fromSession = sessionStorage.getItem(ROOM_STORAGE_KEY);
  console.log("getSavedRoomId: localStorage =", fromLocal, "| sessionStorage =", fromSession);
  return fromLocal || fromSession;
}

function saveRoomId(roomId) {
  if (roomId) {
    try {
      // Сохраняем в localStorage
      localStorage.setItem(ROOM_STORAGE_KEY, roomId);

      // Также сохраняем в sessionStorage как резервную копию
      sessionStorage.setItem(ROOM_STORAGE_KEY, roomId);

      console.log("Комната сохранена в storage:", roomId);
      console.log("Проверка localStorage:", localStorage.getItem(ROOM_STORAGE_KEY));
      console.log("Проверка sessionStorage:", sessionStorage.getItem(ROOM_STORAGE_KEY));

      // Обновляем URL - используем pushState для надёжности
      const currentUrl = new URL(window.location.href);
      if (currentUrl.searchParams.get("room") !== roomId) {
        currentUrl.searchParams.set("room", roomId);
        // Используем replaceState, чтобы не создавать историю
        window.history.replaceState({ roomId: roomId }, "", currentUrl.toString());
        console.log("URL обновлён:", currentUrl.toString());
      }
    } catch (e) {
      console.error("Ошибка сохранения комнаты:", e);
    }
  } else {
    console.warn("saveRoomId вызван с пустым roomId");
  }
}

function clearSavedRoomId() {
  localStorage.removeItem(ROOM_STORAGE_KEY);
  sessionStorage.removeItem(ROOM_STORAGE_KEY);
  // Удаляем параметр room из URL
  const newUrl = new URL(window.location.href);
  newUrl.searchParams.delete("room");
  window.history.replaceState({}, "", newUrl.toString());
  console.log("Сохранённая комната очищена");
}

// Инициализация комнаты
async function initRoom() {
  const participantID = getParticipantID();
  const displayName = getDisplayName();

  // Получение параметров комнаты из URL или localStorage
  const urlParams = new URLSearchParams(window.location.search);
  const roomIdFromUrl = urlParams.get("room");
  const savedRoomId = getSavedRoomId();

  // Диагностика: проверяем все источники ID комнаты
  console.log("=== Диагностика initRoom ===");
  console.log("URL:", window.location.href);
  console.log("search:", window.location.search);
  console.log("roomIdFromUrl:", roomIdFromUrl);
  console.log("savedRoomId (из функции):", savedRoomId);
  console.log("localStorage.current_room_id:", localStorage.getItem("current_room_id"));
  console.log("sessionStorage.current_room_id:", sessionStorage.getItem("current_room_id"));
  console.log("Все ключи localStorage:", JSON.stringify(Object.keys(localStorage)));
  console.log("Все ключи sessionStorage:", JSON.stringify(Object.keys(sessionStorage)));

  // Приоритет: URL параметр > сохранённая комната > создание новой
  const roomId = roomIdFromUrl || savedRoomId;
  console.log("Итоговый roomId:", roomId);
  console.log("Будет создана новая комната:", !roomId);

  // Флаг для отслеживания, была ли комната создана заново
  let isNewlyCreatedRoom = false;

  try {
    roomBanner.textContent = "Подключение к комнате...";
    roomSubtitle.textContent = "Подключаемся...";

    let roomData;
    if (roomId) {
      // Присоединение к существующей комнате (из URL или сохранённой)
      console.log("Присоединение к комнате:", roomId, roomIdFromUrl ? "(из URL)" : "(из localStorage)");
      currentRoomId = roomId;

      try {
        roomData = await joinExistingRoom(roomId, displayName);
        // Сохраняем ID и обновляем URL
        saveRoomId(roomId);
      } catch (joinError) {
        // Если комната не найдена - показываем ошибку и перенаправляем на главную
        console.error("Не удалось присоединиться к комнате:", joinError.message);
        clearSavedRoomId();

        if (joinError.message.includes("не найдена") || joinError.message.includes("not found") || joinError.message.includes("not active")) {
          alert("Комната не найдена или неактивна. Вы будете перенаправлены на главную страницу.");
          window.location.href = "/";
          return;
        } else {
          throw joinError;
        }
      }
    } else {
      // Создание новой комнаты через API
      console.log("Создание новой комнаты...");
      roomData = await createNewRoom(displayName);
      currentRoomId = roomData.id;
      isNewlyCreatedRoom = true;
      if (currentRoomId) {
        console.log("Комната создана, ID:", currentRoomId);
        // Сохраняем ID комнаты и обновляем URL
        saveRoomId(currentRoomId);
      }
    }

    // Получение LiveKit токена (анонимный endpoint)
    const tokenUrl = `${API_BASE}/rooms/${currentRoomId}/media/token`;
    const headers = getAPIHeaders();

    console.log("Requesting LiveKit token:", {
      url: tokenUrl,
      roomId: currentRoomId,
      participantID: getParticipantID(),
      displayName: displayName,
      headers: headers
    });

    const tokenResponse = await fetch(tokenUrl, {
      method: "POST",
      headers: headers,
      body: JSON.stringify({ display_name: displayName }),
    });

    console.log("Token response:", {
      status: tokenResponse.status,
      statusText: tokenResponse.statusText,
      ok: tokenResponse.ok,
      url: tokenResponse.url
    });

    if (!tokenResponse.ok) {
      const errorText = await tokenResponse.text();
      console.error("Ошибка получения токена:", {
        status: tokenResponse.status,
        statusText: tokenResponse.statusText,
        errorText: errorText,
        url: tokenResponse.url,
        requestUrl: tokenUrl,
        headers: headers
      });

      if (tokenResponse.status === 401) {
        throw new Error("Ошибка аутентификации (401). Проверьте, что заголовок X-Participant-ID отправляется правильно. Заголовки: " + JSON.stringify(headers));
      }

      throw new Error("Не удалось получить токен для подключения (статус " + tokenResponse.status + "): " + errorText);
    }

    const tokenData = await tokenResponse.json();
    console.log("Ответ от сервера (tokenData):", {
      hasToken: !!tokenData.token,
      hasUrl: !!tokenData.url,
      tokenLength: tokenData.token ? tokenData.token.length : 0,
      url: tokenData.url,
      allKeys: Object.keys(tokenData)
    });

    let { token: livekitToken, url: livekitUrl } = tokenData;

    // Получаем информацию о сервере для правильного URL
    const info = await getServerInfo();

    // Валидация и нормализация URL
    if (!livekitUrl) {
      console.error("URL не получен от сервера. Полный ответ:", tokenData);
      // Используем URL из информации о сервере
      livekitUrl = info.livekit_url || `ws://${info.host_ip}:${info.livekit_port}`;
      console.warn("Используем URL из server-info:", livekitUrl);
    }

    // Преобразуем URL для работы в локальной сети
    livekitUrl = transformLiveKitUrl(livekitUrl);

    // Проверяем, что URL валидный
    try {
      new URL(livekitUrl);
      console.log("Итоговый LiveKit URL валиден:", livekitUrl);
    } catch (e) {
      console.error("Невалидный URL LiveKit после преобразований:", livekitUrl, e);
      // Используем URL на основе текущего хоста
      const currentHost = window.location.hostname;
      livekitUrl = (window.location.protocol === 'https:' ? 'wss://' + currentHost : 'ws://' + currentHost + ':7880');
      console.warn("Используем fallback URL:", livekitUrl);
    }

    console.log("Подключение к LiveKit:", livekitUrl);

    // Подключение к LiveKit
    // Получаем LiveKit SDK (должен быть уже загружен)
    const LiveKit = getLiveKitSDK();

    if (!LiveKit || !LiveKit.Room) {
      // Детальная диагностика
      console.error("=== ДИАГНОСТИКА LiveKit SDK ===");
      console.error("window.LiveKit:", window.LiveKit);
      console.error("window.LiveKitClient:", window.LiveKitClient);
      console.error("typeof LiveKitClient:", typeof LiveKitClient);
      console.error("typeof LiveKit:", typeof LiveKit);

      const allGlobals = Object.keys(window).filter(k =>
        k.toLowerCase().includes('livekit') ||
        k.toLowerCase().includes('room') ||
        k.toLowerCase().includes('track')
      );
      console.error("Все глобальные переменные с 'livekit', 'room', 'track':", allGlobals);

      // Проверяем, загружен ли скрипт вообще
      const scripts = Array.from(document.querySelectorAll('script')).map(s => s.src);
      console.error("Загруженные скрипты:", scripts);

      throw new Error("LiveKit SDK не загружен. Проверьте консоль браузера для деталей.");
    }

    console.log("Используем LiveKit SDK:", LiveKit);
    console.log("Доступные экспорты в LiveKit:", Object.keys(LiveKit).slice(0, 30));

    // Получаем необходимые классы и функции из SDK
    const Room = LiveKit.Room || (LiveKit.default && LiveKit.default.Room);
    const createLocalAudioTrack = LiveKit.createLocalAudioTrack || (LiveKit.default && LiveKit.default.createLocalAudioTrack);
    const createLocalVideoTrack = LiveKit.createLocalVideoTrack || (LiveKit.default && LiveKit.default.createLocalVideoTrack);
    const createLocalScreenTracks = LiveKit.createLocalScreenTracks || (LiveKit.default && LiveKit.default.createLocalScreenTracks);

    if (!Room) {
      throw new Error("Room класс не найден в LiveKit SDK. Доступные ключи: " + Object.keys(LiveKit).join(", "));
    }

    if (!createLocalAudioTrack || !createLocalVideoTrack) {
      throw new Error("Функции создания треков не найдены в LiveKit SDK");
    }

    // Сохраняем функции для использования в других местах
    window.LiveKitClient = {
      Room,
      createLocalAudioTrack,
      createLocalVideoTrack,
      createLocalScreenTracks,
      DataPacket_Kind: LiveKit.DataPacket_Kind || (LiveKit.default && LiveKit.default.DataPacket_Kind)
    };

    room = new Room();

    // Обработчики событий
    room.on("trackSubscribed", handleTrackSubscribed);
    room.on("trackUnsubscribed", handleTrackUnsubscribed);
    room.on("trackMuted", handleTrackMuted);
    room.on("trackUnmuted", handleTrackUnmuted);
    room.on("dataReceived", handleDataReceived);
    room.on("participantConnected", handleParticipantConnected);
    room.on("participantDisconnected", handleParticipantDisconnected);
    room.on("localTrackPublished", handleLocalTrackPublished);
    room.on("localTrackUnpublished", handleLocalTrackUnpublished);

    // Подключение к комнате
    console.log("Подключение к LiveKit комнате:", {
      url: livekitUrl,
      tokenLength: livekitToken ? livekitToken.length : 0,
      roomName: room ? "уже создан" : "будет создан"
    });

    try {
      await room.connect(livekitUrl, livekitToken);
      console.log("Успешно подключено к LiveKit комнате");
    } catch (connectError) {
      console.error("Ошибка подключения к LiveKit:", connectError);
      console.error("URL:", livekitUrl);
      console.error("Токен (первые 50 символов):", livekitToken ? livekitToken.substring(0, 50) : "отсутствует");
      throw connectError;
    }

    // window.LiveKitClient уже установлен выше, не нужно дублировать

    // Получение медиа-настроек из localStorage
    const micEnabled = localStorage.getItem("micEnabled") === "true";
    const cameraEnabled = localStorage.getItem("cameraEnabled") === "true";

    // Загружаем список устройств перед публикацией треков
    await loadAvailableDevices();

    // Публикация локальных треков
    await publishLocalTracks(micEnabled, cameraEnabled);

    // Инициализируем кнопки прокрутки
    setupScrollButtons();
    
    // Запускаем определение активного говорящего
    startActiveSpeakerDetection();

    // Обновляем layout после подключения
    updateRoomLayout();

    roomBanner.textContent = `Подключено к комнате`;
    roomSubtitle.textContent = `ID: ${currentRoomId}`;

    // Подсказка с ID комнаты на кнопке участников (Удалено по просьбе пользователя)
    const participantsBtn = document.getElementById("room-participants");
    if (participantsBtn && currentRoomId) {
      // participantsBtn.setAttribute("data-tip", `ID комнаты: ${currentRoomId}`);
    }

    // Показываем кнопку копирования ID комнаты
    if (copyRoomIdBtn && currentRoomId) {
      copyRoomIdBtn.style.display = "flex";
    }

    // Однократное модальное окно с ID комнаты для создателя новой комнаты
    if (isNewlyCreatedRoom && currentRoomId) {
      showRoomIdModal(currentRoomId);
    }
  } catch (error) {
    console.error("Room init error:", error);
    roomBanner.textContent = "Ошибка подключения к комнате";
    roomSubtitle.textContent = error.message;
    alert("Не удалось подключиться к комнате: " + error.message);
  }
}

async function publishLocalTracks(audioEnabled, videoEnabled) {
  try {
    const { createLocalAudioTrack, createLocalVideoTrack } = window.LiveKitClient || {};
    if (!createLocalAudioTrack || !createLocalVideoTrack) {
      console.error("LiveKit functions not available");
      return;
    }

    // Загружаем устройства если еще не загружены
    if (availableDevices.videoInputs.length === 0 && availableDevices.audioInputs.length === 0) {
      await loadAvailableDevices();
    }

    if (audioEnabled) {
      try {
        const audioOptions = {
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true,
        };

        // Используем выбранный микрофон если указан
        if (selectedMicrophoneId) {
          audioOptions.deviceId = { exact: selectedMicrophoneId };
        }

        localAudioTrack = await createLocalAudioTrack(audioOptions);
        await room.localParticipant.publishTrack(localAudioTrack);
        isMicEnabled = true;
        updateMicButton();
      } catch (error) {
        console.error("Error creating audio track:", error);
        // Пробуем без указания устройства
        try {
          localAudioTrack = await createLocalAudioTrack({
            echoCancellation: true,
            noiseSuppression: true,
            autoGainControl: true,
          });
          await room.localParticipant.publishTrack(localAudioTrack);
          isMicEnabled = true;
          updateMicButton();
        } catch (retryError) {
          console.error("Error creating audio track (retry):", retryError);
        }
      }
    }

    if (videoEnabled) {
      try {
        const videoOptions = {
          resolution: { width: 1280, height: 720 },
        };

        // Используем выбранную камеру если указана
        if (selectedCameraId) {
          videoOptions.deviceId = { exact: selectedCameraId };
        } else {
          videoOptions.facingMode = "user";
        }

        localVideoTrack = await createLocalVideoTrack(videoOptions);
        await room.localParticipant.publishTrack(localVideoTrack);
        // attachLocalVideoTrack уже не нужен для нового layout, updateRoomLayout все сделает
        // attachLocalVideoTrack(localVideoTrack); 
        isCameraEnabled = true;
        updateCameraButton();
        updateRoomLayout(); // Обновляем layout сразу после создания трека
      } catch (error) {
        console.error("Error creating video track:", error);
        // Пробуем без указания устройства
        try {
          localVideoTrack = await createLocalVideoTrack({
            resolution: { width: 1280, height: 720 },
            facingMode: "user",
          });
          await room.localParticipant.publishTrack(localVideoTrack);
          // attachLocalVideoTrack(localVideoTrack);
          isCameraEnabled = true;
          updateCameraButton();
          updateRoomLayout();
        } catch (retryError) {
          console.error("Error creating video track (retry):", retryError);
        }
      }
    }
  } catch (error) {
    console.error("Error publishing tracks:", error);
  }
}

// Загрузка списка доступных устройств
async function loadAvailableDevices() {
  console.log("Loading available devices...");

  // Проверяем доступность mediaDevices (недоступен на HTTP кроме localhost)
  if (!navigator.mediaDevices || !navigator.mediaDevices.enumerateDevices) {
    console.warn("⚠️ Media devices API недоступен");
    // Показываем уведомление пользователю
    const isSecure = location.protocol === 'https:' || location.hostname === 'localhost' || location.hostname === '127.0.0.1';
    if (!isSecure) {
      showNotification("Камера/микрофон недоступны на HTTP. Вы можете только смотреть.", "warning");
    }
    updateDeviceSelects();
    return null;
  }

  // First, try to enumerate devices without requesting permission
  let devices = await navigator.mediaDevices.enumerateDevices();
  console.log("Initial device list:", devices);

  // Check if we have real device labels (permission was already granted)
  const hasLabels = devices.some(d => d.label && d.label.length > 0);

  if (!hasLabels) {
    console.log("No device labels found, requesting permissions separately...");

    // Request audio permission first
    try {
      console.log("Requesting audio permission...");
      const audioStream = await navigator.mediaDevices.getUserMedia({ audio: true });
      audioStream.getTracks().forEach(track => track.stop());
      console.log("Audio permission granted");
    } catch (audioError) {
      console.warn("Audio permission denied or error:", audioError);
    }

    // Request video permission separately
    try {
      console.log("Requesting video permission...");
      const videoStream = await navigator.mediaDevices.getUserMedia({ video: true });
      videoStream.getTracks().forEach(track => track.stop());
      console.log("Video permission granted");
    } catch (videoError) {
      console.warn("Video permission denied or error:", videoError);
    }

    // Re-enumerate devices now that we (hopefully) have permissions
    console.log("Re-enumerating devices after permission requests...");
    devices = await navigator.mediaDevices.enumerateDevices();
  }

  console.log("Final device list:", devices);

  availableDevices.videoInputs = devices.filter(device => device.kind === 'videoinput');
  availableDevices.audioInputs = devices.filter(device => device.kind === 'audioinput');
  availableDevices.audioOutputs = devices.filter(device => device.kind === 'audiooutput');

  console.log("Доступные устройства:", {
    камеры: availableDevices.videoInputs.map(d => d.label || 'No label'),
    микрофоны: availableDevices.audioInputs.map(d => d.label || 'No label'),
    динамики: availableDevices.audioOutputs.map(d => d.label || 'No label')
  });

  // Загружаем сохраненные настройки
  selectedCameraId = localStorage.getItem("selectedCameraId") || null;
  selectedMicrophoneId = localStorage.getItem("selectedMicrophoneId") || null;
  selectedSpeakerId = localStorage.getItem("selectedSpeakerId") || null;

  // Обновляем селекты в модальном окне
  updateDeviceSelects();

  return availableDevices;
}

// Обновление селектов в модальном окне
function updateDeviceSelects() {
  // Очищаем селекты
  cameraSelect.innerHTML = '';
  microphoneSelect.innerHTML = '';
  speakerSelect.innerHTML = '';

  // Фильтруем устройства, исключая "default" (так как они дублируют реальные устройства)
  const videoInputs = availableDevices.videoInputs.filter(d => d.deviceId !== 'default');
  const audioInputs = availableDevices.audioInputs.filter(d => d.deviceId !== 'default');
  const audioOutputs = availableDevices.audioOutputs.filter(d => d.deviceId !== 'default');

  // Заполняем камеры
  if (videoInputs.length === 0) {
    cameraSelect.innerHTML = '<option value="">Камеры не найдены</option>';
  } else {
    videoInputs.forEach((device, index) => {
      const option = document.createElement('option');
      option.value = device.deviceId;
      option.textContent = device.label || `Камера ${index + 1}`;
      if (device.deviceId === selectedCameraId) {
        option.selected = true;
      }
      cameraSelect.appendChild(option);
    });
  }

  // Заполняем микрофоны
  if (audioInputs.length === 0) {
    microphoneSelect.innerHTML = '<option value="">Микрофоны не найдены</option>';
  } else {
    audioInputs.forEach((device, index) => {
      const option = document.createElement('option');
      option.value = device.deviceId;
      option.textContent = device.label || `Микрофон ${index + 1}`;
      if (device.deviceId === selectedMicrophoneId) {
        option.selected = true;
      }
      microphoneSelect.appendChild(option);
    });
  }

  // Заполняем динамики
  if (audioOutputs.length === 0) {
    speakerSelect.innerHTML = '<option value="">Динамики не найдены</option>';
  } else {
    audioOutputs.forEach((device, index) => {
      const option = document.createElement('option');
      option.value = device.deviceId;
      option.textContent = device.label || `Динамики ${index + 1}`;
      if (device.deviceId === selectedSpeakerId) {
        option.selected = true;
      }
      speakerSelect.appendChild(option);
    });
  }
}

// Показать модальное окно выбора устройств
async function showDeviceSettingsModal() {
  // Show the modal immediately
  deviceSettingsModal.classList.remove("hidden");

  // Set loading state
  cameraSelect.innerHTML = '<option value="">Загрузка...</option>';
  microphoneSelect.innerHTML = '<option value="">Загрузка...</option>';
  speakerSelect.innerHTML = '<option value="">Загрузка...</option>';

  // Always refresh devices when opening the modal
  try {
    await loadAvailableDevices();
  } catch (error) {
    console.error("Error loading devices:", error);
    // Show error state in selects
    cameraSelect.innerHTML = '<option value="">Ошибка загрузки</option>';
    microphoneSelect.innerHTML = '<option value="">Ошибка загрузки</option>';
    speakerSelect.innerHTML = '<option value="">Ошибка загрузки</option>';
  }
}

// Скрыть модальное окно выбора устройств
function hideDeviceSettingsModal() {
  deviceSettingsModal.classList.add("hidden");
}

// Применить выбранные устройства
async function applyDeviceSettings() {
  const newCameraId = cameraSelect.value || null;
  const newMicrophoneId = microphoneSelect.value || null;
  const newSpeakerId = speakerSelect.value || null;

  // Сохраняем выбор
  selectedCameraId = newCameraId;
  selectedMicrophoneId = newMicrophoneId;
  selectedSpeakerId = newSpeakerId;

  localStorage.setItem("selectedCameraId", selectedCameraId || "");
  localStorage.setItem("selectedMicrophoneId", selectedMicrophoneId || "");
  localStorage.setItem("selectedSpeakerId", selectedSpeakerId || "");

  console.log("Применены настройки устройств:", {
    камера: selectedCameraId,
    микрофон: selectedMicrophoneId,
    динамики: selectedSpeakerId
  });

  // Если треки уже созданы, пересоздаем их с новыми устройствами
  if (room) {
    const wasMicEnabled = isMicEnabled;
    const wasCameraEnabled = isCameraEnabled;

    // Останавливаем текущие треки
    if (localAudioTrack) {
      await room.localParticipant.unpublishTrack(localAudioTrack);
      localAudioTrack.stop();
      localAudioTrack = null;
      isMicEnabled = false;
    }

    if (localVideoTrack) {
      await room.localParticipant.unpublishTrack(localVideoTrack);
      localVideoTrack.stop();
      localVideoTrack = null;
      detachLocalVideoTrack();
      isCameraEnabled = false;
    }

    // Создаем новые треки с выбранными устройствами
    await publishLocalTracks(wasMicEnabled, wasCameraEnabled);

    // Применяем выбранные динамики к существующим аудио элементам
    if (selectedSpeakerId) {
      const audioElements = document.querySelectorAll('audio[data-participant]');
      for (const audioElement of audioElements) {
        if (audioElement.setSinkId) {
          try {
            await audioElement.setSinkId(selectedSpeakerId);
            console.log("Применены динамики к существующему аудио элементу");
          } catch (sinkError) {
            console.warn("Не удалось применить динамики:", sinkError);
          }
        }
      }
    }
  }

  hideDeviceSettingsModal();
}

function attachLocalVideoTrack(track) {
  if (!track) return;

  // Прикрепляем к статическим элементам для разных видов
  if (localVideoElement) {
    track.attach(localVideoElement);
  }
  if (localVideoChatElement) {
    track.attach(localVideoChatElement);
  }
  if (localVideoScreenElement) {
    track.attach(localVideoScreenElement);
  }

  // Также прикрепляем к динамически созданному элементу локального участника
  if (room && room.localParticipant) {
    const localParticipantEl = document.querySelector(`[data-participant-id="${room.localParticipant.identity}"].room-participant--me`);
    if (localParticipantEl) {
      const videoEl = localParticipantEl.querySelector("video");
      if (videoEl) {
        track.attach(videoEl);
        videoEl.style.display = "block";
        const avatarEl = localParticipantEl.querySelector(".participant-avatar");
        if (avatarEl) avatarEl.style.display = "none";
      }
    }
  }
}

function detachLocalVideoTrack() {
  if (localVideoElement) {
    localVideoElement.srcObject = null;
  }
  if (localVideoChatElement) {
    localVideoChatElement.srcObject = null;
  }
  if (localVideoScreenElement) {
    localVideoScreenElement.srcObject = null;
  }
}

function attachVideoTrack(track, elementId) {
  const element = document.getElementById(elementId);
  if (element) {
    track.attach(element);
    console.log(`Attached track ${track.kind} to ${elementId}`);
    return true;
  }
  console.warn(`Element ${elementId} not found for attaching track`);
  return false;
}

async function handleTrackSubscribed(track, publication, participant) {
  console.log("Track subscribed:", track.kind, "from", participant.identity);

  // Игнорируем локальные треки
  if (participant.isLocal) {
    return;
  }

  // Получаем или создаем запись для участника
  let participantData = remoteParticipants.get(participant.identity);
  if (!participantData) {
    participantData = {
      videoTrack: null,
      audioTrack: null,
      element: null,
    };
    remoteParticipants.set(participant.identity, participantData);
    
    // Сразу обновляем layout чтобы показать нового участника
    updateRoomLayout();
  }

  if (track.kind === "video") {
    participantData.videoTrack = track;

    // Проверяем, является ли трек демонстрацией экрана
    if (track.source === "screen_share" || track.source === "screen") {
      console.log("Detected screen share track from", participant.identity);
      activeScreenShareId = participant.identity;
      
      // Если это не вид с демонстрацией экрана, переключаемся на него
      if (currentRoomView !== "screen") {
        const screenElement = document.getElementById("screen-share-video");
        if (screenElement) {
          track.attach(screenElement);
          setRoomView("screen");
          console.log("Attached screen share track to screen-share-video and switched to screen view");
        }
      }
      
      // Обновляем layout для отображения демонстрирующего в центре
      updateRoomLayout();
    } else {
      // Используем динамически созданные элементы
      updateRoomLayout();

      // Ждем немного чтобы элементы создались
      setTimeout(() => {
        const participantEl = document.querySelector(`[data-participant-id="${participant.identity}"]`);
        let videoElement;

        if (participantEl) {
          videoElement = participantEl.querySelector("video");
        }

        if (videoElement) {
          participantData.element = videoElement;
          track.attach(videoElement);
          videoElement.style.display = "block";
          videoElement.autoplay = true;
          videoElement.playsInline = true;

          // Скрываем иконку аватара
          const avatarEl = participantEl.querySelector(".participant-avatar");
          if (avatarEl) avatarEl.style.display = "none";

          videoElement.play().catch(e => console.warn("Autoplay blocked:", e));
          console.log("✅ Attached video track to participant", participant.identity);
        } else {
          console.error("❌ Video element not found for participant:", participant.identity);
        }
      }, 100);
    }
  } else if (track.kind === "audio") {
    participantData.audioTrack = track;

    // Создаем аудио элемент для воспроизведения
    const audioElement = document.createElement("audio");
    audioElement.autoplay = true;
    audioElement.playsInline = true;
    audioElement.setAttribute("data-participant", participant.identity);

    // Применяем выбранные динамики если указаны
    if (selectedSpeakerId && audioElement.setSinkId) {
      try {
        await audioElement.setSinkId(selectedSpeakerId);
        console.log("Применены динамики для участника:", participant.identity);
      } catch (sinkError) {
        console.warn("Не удалось применить динамики:", sinkError);
      }
    }

    track.attach(audioElement);
    document.body.appendChild(audioElement);
    console.log("Attached audio track for", participant.identity);

    // Создаем анализатор аудио для определения активного говорящего
    // Используем setTimeout чтобы дать время треку подключиться к элементу
    setTimeout(() => {
      try {
        // Получаем MediaStream из трека
        const mediaStream = track.mediaStreamTrack ? new MediaStream([track.mediaStreamTrack]) : null;
        if (!mediaStream) {
          console.warn("No media stream available for analyser");
          return;
        }

        const audioContext = new (window.AudioContext || window.webkitAudioContext)();
        const source = audioContext.createMediaStreamSource(mediaStream);
        const analyser = audioContext.createAnalyser();
        analyser.fftSize = 256;
        analyser.smoothingTimeConstant = 0.8;
        source.connect(analyser);

        audioAnalysers.set(participant.identity, {
          audioContext: audioContext,
          analyser: analyser,
          dataArray: new Uint8Array(analyser.frequencyBinCount)
        });

        console.log("Audio analyser created for", participant.identity);
        
        // Запускаем определение активного говорящего если еще не запущено
        startActiveSpeakerDetection();
      } catch (error) {
        console.warn("Failed to create audio analyser:", error);
      }
    }, 500);
  }

  // Обновляем layout после подписки на трек
  updateRoomLayout();
}

function handleTrackUnsubscribed(track, publication, participant) {
  console.log("Track unsubscribed:", track.kind, "from", participant.identity);

  if (participant.isLocal) {
    return;
  }

  const participantData = remoteParticipants.get(participant.identity);
  if (participantData) {
    if (track.kind === "video") {
      if (participantData.element) {
        track.detach(participantData.element);
        participantData.element.srcObject = null;
      }
      participantData.videoTrack = null;
    } else if (track.kind === "audio") {
      const audioElement = document.querySelector(`audio[data-participant="${participant.identity}"]`);
      if (audioElement) {
        track.detach(audioElement);
        audioElement.remove();
      }
      
      // Удаляем анализатор аудио
      const analyserData = audioAnalysers.get(participant.identity);
      if (analyserData && analyserData.audioContext) {
        analyserData.audioContext.close().catch(console.warn);
        audioAnalysers.delete(participant.identity);
      }
      
      participantData.audioTrack = null;
    }

    // Если отписались от трека демонстрации экрана, возвращаемся в обычный вид
    if (track.source === "screen_share" || track.source === "screen") {
      console.log("Screen share track unsubscribed, returning to default view");
      if (activeScreenShareId === participant.identity) {
        activeScreenShareId = null;
      }
      setRoomView("default");
    }

    // Если у участника больше нет треков, удаляем его из списка
    if (!participantData.videoTrack && !participantData.audioTrack) {
      remoteParticipants.delete(participant.identity);
    }

    // Обновляем layout после отписки от трека
    updateRoomLayout();
  }
}

function handleTrackMuted(publication, participant) {
  console.log("Track muted:", publication.kind, "from", participant.identity);
}

function handleTrackUnmuted(publication, participant) {
  console.log("Track unmuted:", publication.kind, "from", participant.identity);
}

function handleLocalTrackPublished(publication, participant) {
  console.log("Local track published:", publication.kind);
  updateRoomLayout();
}

function handleLocalTrackUnpublished(publication, participant) {
  console.log("Local track unpublished:", publication.kind);
  updateRoomLayout();
}

function handleDataReceived(payload, participant) {
  try {
    const data = JSON.parse(new TextDecoder().decode(payload));
    if (data.type === "chat") {
      appendChatMessage({
        author: participant.name || "Участник",
        text: data.message,
        me: false,
      });
    }
  } catch (error) {
    console.error("Error parsing data:", error);
  }
}

function handleParticipantConnected(participant) {
  console.log("Participant connected:", participant.identity, participant.name);
  appendChatMessage({
    author: "Система",
    text: `${participant.name || participant.identity} присоединился`,
    me: false,
  });

  // Создаем запись для нового участника если её еще нет
  if (!remoteParticipants.has(participant.identity)) {
    remoteParticipants.set(participant.identity, {
      videoTrack: null,
      audioTrack: null,
      element: null,
    });
  }

  // Обновляем layout при подключении участника
  updateRoomLayout();
}

function handleParticipantDisconnected(participant) {
  console.log("Participant disconnected:", participant.identity);

  // Очищаем треки участника
  const participantData = remoteParticipants.get(participant.identity);
  if (participantData) {
    if (participantData.videoTrack && participantData.element) {
      participantData.videoTrack.detach(participantData.element);
      participantData.element.srcObject = null;
    }
    const audioElement = document.querySelector(`audio[data-participant="${participant.identity}"]`);
    if (audioElement) {
      audioElement.remove();
    }
    remoteParticipants.delete(participant.identity);
  }

  // Удаляем элемент участника из DOM
  const participantEl = document.querySelector(`[data-participant-id="${participant.identity}"]`);
  if (participantEl && !participantEl.classList.contains("room-participant--me")) {
    participantEl.remove();
  }

  appendChatMessage({
    author: "Система",
    text: `${participant.name || participant.identity} покинул комнату`,
    me: false,
  });

  // Обновляем layout при отключении участника
  updateRoomLayout();
}

function getRemoteVideoElementId() {
  switch (currentRoomView) {
    case "chat":
      return "remote-video-chat";
    case "screen":
      return "remote-video-screen";
    default:
      return "remote-video";
  }
}

function setRoomView(mode) {
  currentRoomView = mode;
  viewDefault.classList.toggle("hidden", mode !== "default");
  viewChat.classList.toggle("hidden", mode !== "chat");
  viewScreen.classList.toggle("hidden", mode !== "screen");

  btnRoomChat.classList.toggle("small-rect--active", mode === "chat");
  btnRoomScreen.classList.toggle("circle-btn--screen-share-on", mode === "screen");

  // Переподключаем удаленные видео треки к новым элементам
  remoteParticipants.forEach((participantData, identity) => {
    if (participantData.videoTrack) {
      const newElementId = getRemoteVideoElementId();
      const newElement = document.getElementById(newElementId);
      if (newElement && participantData.element !== newElement) {
        participantData.videoTrack.detach(participantData.element);
        participantData.videoTrack.attach(newElement);
        participantData.element = newElement;
      }
    }
  });

  // Обновляем layout после смены вида
  updateRoomLayout();
}

// Функция для создания элемента участника
function createParticipantElement(participantIdentity, participantName, isLocal = false) {
  const participantDiv = document.createElement("div");
  participantDiv.className = `room-participant ${isLocal ? 'room-participant--me' : ''} glass-tile`;
  participantDiv.setAttribute("data-participant-id", participantIdentity);

  const video = document.createElement("video");
  video.id = `video-${participantIdentity}`;
  video.autoplay = true;
  video.playsInline = true;
  video.muted = isLocal;

  const nameDiv = document.createElement("div");
  nameDiv.className = "room-name";
  nameDiv.textContent = participantName || (isLocal ? "Я" : "Участник");
  nameDiv.id = `name-${participantIdentity}`;

  // Иконка для участника без видео
  const avatarIcon = document.createElement("div");
  avatarIcon.className = "participant-avatar";
  avatarIcon.innerHTML = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
    <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
    <circle cx="12" cy="7" r="4" />
  </svg>`;

  // Индикатор микрофона
  const micIndicator = document.createElement("div");
  micIndicator.className = "participant-mic-indicator";
  micIndicator.innerHTML = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
    <path d="M12 14a3 3 0 0 0 3-3V6a3 3 0 1 0-6 0v5a3 3 0 0 0 3 3Z" />
    <path d="M19 10v2a7 7 0 0 1-14 0v-2" />
    <line x1="12" y1="18" x2="12" y2="22" />
  </svg>`;

  participantDiv.appendChild(video);
  participantDiv.appendChild(avatarIcon);
  participantDiv.appendChild(micIndicator);
  participantDiv.appendChild(nameDiv);

  return { container: participantDiv, video, nameDiv, avatarIcon, micIndicator };
}

// Определение активного говорящего через анализ аудио
function detectActiveSpeaker() {
  if (!room) {
    return null;
  }
  
  // Разрешаем определение в default и chat режимах
  if (currentRoomView !== "default" && currentRoomView !== "chat") {
    return null;
  }

  let maxVolume = 0;
  let activeId = null;

  // Проверяем локального участника
  if (localAudioTrack && isMicEnabled && !localAudioTrack.isMuted) {
    // Для локального участника используем простую проверку (можно улучшить)
    activeId = room.localParticipant.identity;
    maxVolume = 0.5; // Базовый уровень для локального участника
  }

  // Проверяем удаленных участников
  remoteParticipants.forEach((participantData, identity) => {
    if (!participantData.audioTrack || participantData.audioTrack.isMuted) {
      return;
    }

    const analyserData = audioAnalysers.get(identity);
    if (analyserData && analyserData.analyser) {
      const dataArray = new Uint8Array(analyserData.analyser.frequencyBinCount);
      analyserData.analyser.getByteFrequencyData(dataArray);
      
      // Вычисляем средний уровень громкости
      let sum = 0;
      for (let i = 0; i < dataArray.length; i++) {
        sum += dataArray[i];
      }
      const average = sum / dataArray.length;
      const volume = average / 255; // Нормализуем от 0 до 1

      if (volume > maxVolume && volume > 0.1) { // Порог активации
        maxVolume = volume;
        activeId = identity;
      }
    }
  });

  // Проверяем демонстрацию экрана
  if (activeScreenShareId) {
    activeId = activeScreenShareId;
  }

  return activeId;
}

// Обновление layout с новым дизайном
function updateRoomLayout() {
  if (!room) return;

  const topList = document.getElementById("participants-top-list");
  const mainParticipant = document.getElementById("room-main-participant");
  const chatColumn = document.getElementById("chat-participants-column");

  // Проверяем режим и наличие контейнеров
  if (currentRoomView === "default") {
    if (!topList || !mainParticipant) return;
  } else if (currentRoomView === "chat") {
    if (!chatColumn) return;
  } else if (currentRoomView !== "screen") {
    return;
  }

  // Определяем активного говорящего
  const newActiveSpeaker = detectActiveSpeaker();
  if (newActiveSpeaker !== activeSpeakerId) {
    activeSpeakerId = newActiveSpeaker;
  }

  // Получаем всех участников (локальный + удаленные)
  const allParticipants = [];
  
  // Добавляем локального участника
  const localName = room.localParticipant.name || localStorage.getItem("display_name") || "Я";
  const localParticipantObj = {
    identity: room.localParticipant.identity,
    name: localName,
    isLocal: true,
    participantData: {
      videoTrack: localVideoTrack,
      audioTrack: localAudioTrack,
      element: null
    }
  };
  allParticipants.push(localParticipantObj);

  // Добавляем удаленных участников
  remoteParticipants.forEach((participantData, identity) => {
    const participant = room.remoteParticipants.get(identity);
    if (participant) {
      allParticipants.push({
        identity: identity,
        name: participant.name || participant.identity || "Участник",
        isLocal: false,
        participantData: participantData
      });
    }
  });

  // Логика рендеринга в зависимости от вида
  if (currentRoomView === "chat") {
    // === CHAT VIEW ===
    chatColumn.innerHTML = "";

    // 1. Local Participant (Me) - Large
    const meData = createParticipantElement(localParticipantObj.identity, localParticipantObj.name, true);
    meData.container.classList.add("participant-me-large");
    setupParticipantElement(meData, localParticipantObj.participantData);
    chatColumn.appendChild(meData.container);

    // 2. Remote Participant (Active or First) - Small
    let remoteToShow = null;
    // Приоритет: активный говорящий (если он не локальный)
    if (activeSpeakerId && activeSpeakerId !== room.localParticipant.identity) {
      remoteToShow = allParticipants.find(p => p.identity === activeSpeakerId);
    }
    // Иначе первый попавшийся удаленный
    if (!remoteToShow) {
      remoteToShow = allParticipants.find(p => !p.isLocal);
    }

    if (remoteToShow) {
      const guestData = createParticipantElement(remoteToShow.identity, remoteToShow.name, false);
      guestData.container.classList.add("participant-guest-small");
      setupParticipantElement(guestData, remoteToShow.participantData);
      chatColumn.appendChild(guestData.container);
    }

  } else if (currentRoomView === "default") {
    // === DEFAULT VIEW ===
    topList.innerHTML = "";
    mainParticipant.innerHTML = "";

    // Определяем, кто будет в центре (активный говорящий или демонстрирующий)
    let centerParticipant = null;
    if (activeScreenShareId) {
      centerParticipant = allParticipants.find(p => p.identity === activeScreenShareId);
    } else if (activeSpeakerId) {
      centerParticipant = allParticipants.find(p => p.identity === activeSpeakerId);
    } else if (allParticipants.length > 0) {
      // Если нет активного говорящего, показываем первого участника
      centerParticipant = allParticipants[0];
    }

    // Создаем элементы для верхней панели (все участники кроме центрального)
    allParticipants.forEach(participant => {
      if (centerParticipant && participant.identity === centerParticipant.identity) {
        return; // Пропускаем центрального участника
      }

      const data = createParticipantElement(participant.identity, participant.name, participant.isLocal);
      
      // Добавляем обработчик клика для переключения на центральную позицию
      data.container.addEventListener("click", () => {
        console.log("Clicked participant:", participant.identity);
        activeSpeakerId = participant.identity;
        updateRoomLayout();
      });

      setupParticipantElement(data, participant.participantData);
      topList.appendChild(data.container);
    });

    // Создаем центрального участника
    if (centerParticipant) {
      const data = createParticipantElement(centerParticipant.identity, centerParticipant.name, centerParticipant.isLocal);
      
      // Добавляем класс для активного говорящего
      if (activeSpeakerId === centerParticipant.identity || activeScreenShareId === centerParticipant.identity) {
        data.container.classList.add("active-speaker");
      }

      setupParticipantElement(data, centerParticipant.participantData);
      mainParticipant.appendChild(data.container);
    }

    // Обновляем кнопки прокрутки
    updateScrollButtons();
  }
  
  console.log(`Layout updated (${currentRoomView}): ${allParticipants.length} participants`);
}

// Helper: Настройка элемента участника (видео, аватар, микрофон)
function setupParticipantElement(data, participantData) {
    const { container, video, avatarIcon, micIndicator } = data;
    const videoEl = video;

    // Усиленная проверка наличия активного видео трека
    const hasVideo = participantData.videoTrack && 
                     participantData.videoTrack.mediaStreamTrack && 
                     participantData.videoTrack.mediaStreamTrack.readyState !== 'ended';

    // Обрабатываем видео
    if (hasVideo) {
      // Отсоединяем от старого элемента если нужно
      if (participantData.element && participantData.element !== videoEl) {
        try {
            participantData.videoTrack.detach(participantData.element);
        } catch(e) { console.warn("Detach error", e); }
      }
      participantData.videoTrack.attach(videoEl);
      videoEl.style.display = "block";
      avatarIcon.style.display = "none";
      participantData.element = videoEl;
      container.classList.remove("no-video");
    } else {
      videoEl.style.display = "none";
      videoEl.srcObject = null; // Явно очищаем источник
      avatarIcon.style.display = "flex";
      participantData.element = null;
      container.classList.add("no-video");
    }

    // Обновляем индикатор микрофона
    const hasAudio = participantData.audioTrack && !participantData.audioTrack.isMuted;
    micIndicator.style.display = hasAudio ? "flex" : "none";
}

// Обновление кнопок прокрутки
function updateScrollButtons() {
  const topList = document.getElementById("participants-top-list");
  const scrollLeftBtn = document.getElementById("scroll-left-btn");
  const scrollRightBtn = document.getElementById("scroll-right-btn");

  if (!topList || !scrollLeftBtn || !scrollRightBtn) {
    return;
  }

  const canScrollLeft = topList.scrollLeft > 0;
  const canScrollRight = topList.scrollLeft < (topList.scrollWidth - topList.clientWidth);

  scrollLeftBtn.disabled = !canScrollLeft;
  scrollRightBtn.disabled = !canScrollRight;
}

// Обработчики прокрутки
function setupScrollButtons() {
  const topList = document.getElementById("participants-top-list");
  const scrollLeftBtn = document.getElementById("scroll-left-btn");
  const scrollRightBtn = document.getElementById("scroll-right-btn");

  if (!topList || !scrollLeftBtn || !scrollRightBtn) {
    return;
  }

  scrollLeftBtn.addEventListener("click", () => {
    topList.scrollBy({ left: -200, behavior: "smooth" });
  });

  scrollRightBtn.addEventListener("click", () => {
    topList.scrollBy({ left: 200, behavior: "smooth" });
  });

  topList.addEventListener("scroll", updateScrollButtons);
  
  // Обновляем кнопки при изменении размера окна
  window.addEventListener("resize", updateScrollButtons);
  
  // Начальное обновление
  setTimeout(updateScrollButtons, 100);
}

// Интервал для обновления активного говорящего
let activeSpeakerUpdateInterval = null;

function startActiveSpeakerDetection() {
  if (activeSpeakerUpdateInterval) {
    return;
  }
  
  activeSpeakerUpdateInterval = setInterval(() => {
    if (room && (currentRoomView === "default" || currentRoomView === "chat")) {
      const newActiveSpeaker = detectActiveSpeaker();
      if (newActiveSpeaker !== activeSpeakerId) {
        activeSpeakerId = newActiveSpeaker;
        updateRoomLayout();
      }
    }
  }, 500); // Проверяем каждые 500мс
}

function stopActiveSpeakerDetection() {
  if (activeSpeakerUpdateInterval) {
    clearInterval(activeSpeakerUpdateInterval);
    activeSpeakerUpdateInterval = null;
  }
}

// Управление микрофоном
btnRoomMic.addEventListener("click", async () => {
  if (!room) return;

  try {
    const { createLocalAudioTrack } = window.LiveKitClient || {};
    if (!createLocalAudioTrack) {
      console.error("LiveKit functions not available");
      return;
    }

    if (isMicEnabled) {
      if (localAudioTrack) {
        await room.localParticipant.unpublishTrack(localAudioTrack);
        localAudioTrack.stop();
        localAudioTrack = null;
      }
      isMicEnabled = false;
    } else {
      const audioOptions = {
        echoCancellation: true,
        noiseSuppression: true,
        autoGainControl: true,
      };

      // Используем выбранный микрофон если указан
      if (selectedMicrophoneId) {
        audioOptions.deviceId = { exact: selectedMicrophoneId };
      }

      try {
        localAudioTrack = await createLocalAudioTrack(audioOptions);
        await room.localParticipant.publishTrack(localAudioTrack);
        isMicEnabled = true;
      } catch (deviceError) {
        // Если не удалось с выбранным устройством, пробуем без него
        console.warn("Ошибка с выбранным микрофоном, пробуем без указания устройства:", deviceError);
        localAudioTrack = await createLocalAudioTrack({
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true,
        });
        await room.localParticipant.publishTrack(localAudioTrack);
        isMicEnabled = true;
      }
    }
    updateMicButton();
  } catch (error) {
    console.error("Error toggling mic:", error);
    alert("Не удалось включить микрофон. Проверьте разрешения браузера.");
  }
});

function updateMicButton() {
  btnRoomMic.classList.toggle("active", isMicEnabled);
}

// Управление камерой
btnRoomCamera.addEventListener("click", async () => {
  if (!room) return;

  try {
    const { createLocalVideoTrack } = window.LiveKitClient || {};
    if (!createLocalVideoTrack) {
      console.error("LiveKit functions not available");
      return;
    }

    if (isCameraEnabled) {
      // Сначала сохраняем ссылку на трек
      const trackToUnpublish = localVideoTrack;
      // Обнуляем глобальную переменную СРАЗУ, чтобы updateRoomLayout увидел изменения
      localVideoTrack = null;
      isCameraEnabled = false;
      // Явно обновляем layout после изменения состояния (сразу, не дожидаясь unpublish)
      updateRoomLayout();
      updateCameraButton();

      if (trackToUnpublish) {
        try {
          await room.localParticipant.unpublishTrack(trackToUnpublish);
        } catch (e) {
          console.warn("Error unpublishing track:", e);
        }
        
        trackToUnpublish.stop();
        detachLocalVideoTrack();
      }
    } else {
      const videoOptions = {
        resolution: { width: 1280, height: 720 },
      };

      // Используем выбранную камеру если указана
      if (selectedCameraId) {
        videoOptions.deviceId = { exact: selectedCameraId };
      } else {
        videoOptions.facingMode = "user";
      }

      try {
        localVideoTrack = await createLocalVideoTrack(videoOptions);
        await room.localParticipant.publishTrack(localVideoTrack);
        isCameraEnabled = true;
        updateCameraButton();
        updateRoomLayout();
      } catch (deviceError) {
        // Если не удалось с выбранной камерой, пробуем без нее
        console.warn("Ошибка с выбранной камерой, пробуем без указания устройства:", deviceError);
        localVideoTrack = await createLocalVideoTrack({
          resolution: { width: 1280, height: 720 },
          facingMode: "user",
        });
        await room.localParticipant.publishTrack(localVideoTrack);
        isCameraEnabled = true;
        updateCameraButton();
        updateRoomLayout();
      }
    }
    updateCameraButton();
  } catch (error) {
    console.error("Error toggling camera:", error);
    alert("Не удалось включить камеру. Проверьте разрешения браузера.");
  }
});

function updateCameraButton() {
  btnRoomCamera.classList.toggle("active", isCameraEnabled);
}

// Демонстрация экрана (server-side screen sharing)
btnRoomScreen.addEventListener("click", async () => {
  if (isScreenSharing) {
    await stopAllScreenSharing();
  } else {
    // Начало демонстрации экрана через сервер
    try {
      await startServerScreenShare();
      isScreenSharing = true;
      setRoomView("screen");
    } catch (error) {
      console.error("Error starting server screen share:", error);
      const errorMessage = error.message || error.toString();

      // If server-side capture fails with "no display found", it's likely not an error we want to alert the user about
      // as it might be a common state or a form of cancellation.
      if (errorMessage.includes("no display found")) {
        console.log("Server-side screen capture not available, trying LiveKit fallback...");
      } else if (errorMessage.includes("room not found") || errorMessage.includes("authentication") || errorMessage.includes("unauthorized")) {
        alert("Ошибка: " + errorMessage);
        return;
      }

      // Fallback to LiveKit screen sharing if server-side fails
      try {
        await startLiveKitScreenShare();
        isScreenSharing = true;
        setRoomView("screen");
      } catch (fallbackError) {
        console.error("Error with LiveKit screen share fallback:", fallbackError);
        const fallbackMsg = fallbackError.message || fallbackError.toString();

        // Suppress alert if user cancelled the permission request
        const isCancellation =
          fallbackError.name === 'NotAllowedError' ||
          fallbackMsg.includes("Permission denied") ||
          fallbackMsg.includes("cancelled");

        if (!isCancellation) {
          alert("Не удалось начать демонстрацию экрана: " + fallbackMsg);
        } else {
          console.log("Screen share cancelled by user.");
        }

        await stopAllScreenSharing(); // Ensure cleanup on total failure
      }
    }
  }
});

// Server-side screen sharing functions
async function startServerScreenShare() {
  if (serverScreenSharePC) {
    console.warn("Server screen share already active");
    return;
  }

  // Create WebRTC peer connection
  serverScreenSharePC = new RTCPeerConnection({
    iceServers: [{ urls: "stun:stun.l.google.com:19302" }],
  });

  // Handle connection state changes
  serverScreenSharePC.onconnectionstatechange = () => {
    const state = serverScreenSharePC.connectionState;
    console.log("Peer connection state:", state);
    if (state === "failed" || state === "disconnected") {
      console.error("Peer connection failed or disconnected:", state);
      // Clean up on failure
      if (serverScreenSharePC) {
        serverScreenSharePC.close();
        serverScreenSharePC = null;
      }
    }
  };

  // Handle errors
  serverScreenSharePC.onerror = (error) => {
    console.error("Peer connection error:", error);
  };

  // Handle incoming video stream from server
  serverScreenSharePC.ontrack = (event) => {
    console.log("Received track from server", event.track.kind);
    if (event.track.kind === "video") {
      // Listen for track ending (e.g. server stops capture)
      event.track.onended = () => {
        console.log("Server screen share track ended");
        stopAllScreenSharing();
      };

      const videoElement = document.getElementById("screen-share-video");
      if (videoElement) {
        videoElement.srcObject = new MediaStream([event.track]);
        videoElement.play().catch((err) => {
          console.error("Error playing video:", err);
        });
      }
    }
  };

  // Handle ICE candidates
  serverScreenSharePC.onicecandidate = async (event) => {
    if (event.candidate && serverScreenSharePeerId) {
      try {
        await fetch(`/screen-share/ice/${serverScreenSharePeerId}`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(event.candidate.toJSON()),
        });
      } catch (error) {
        console.error("Error sending ICE candidate:", error);
      }
    }
  };

  // Create offer
  try {
    const offer = await serverScreenSharePC.createOffer();
    await serverScreenSharePC.setLocalDescription(offer);

    // Send offer to server
    const response = await fetch("/screen-share/offer", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        sdp: offer.sdp,
        type: offer.type,
      }),
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      const errorMessage = errorData.error || "Failed to establish connection with server";
      throw new Error(errorMessage);
    }

    const answerData = await response.json();
    serverScreenSharePeerId = response.headers.get("X-Peer-Connection-ID");

    // Set remote description (answer)
    await serverScreenSharePC.setRemoteDescription(
      new RTCSessionDescription({
        type: answerData.type,
        sdp: answerData.sdp,
      })
    );

    // Poll for server ICE candidates
    if (serverScreenSharePeerId) {
      pollServerICECandidates(serverScreenSharePeerId);
    }
  } catch (error) {
    console.error("Error in startServerScreenShare:", error);
    if (serverScreenSharePC) {
      serverScreenSharePC.close();
      serverScreenSharePC = null;
    }
    throw error;
  }
}

async function pollServerICECandidates(peerId) {
  const maxAttempts = 50;
  let attempts = 0;

  const poll = async () => {
    if (attempts >= maxAttempts || !serverScreenSharePC) {
      return;
    }

    try {
      const response = await fetch(`/screen-share/ice/${peerId}`);
      if (response.ok) {
        const data = await response.json();
        if (data.candidates && data.candidates.length > 0) {
          for (const candidate of data.candidates) {
            await serverScreenSharePC.addICECandidate(candidate);
          }
        }
      }
    } catch (error) {
      console.error("Error polling ICE candidates:", error);
    }

    attempts++;
    setTimeout(poll, 500);
  };

  poll();
}

async function stopServerScreenShare() {
  if (serverScreenSharePeerId) {
    try {
      await fetch(`/screen-share/hangup/${serverScreenSharePeerId}`, {
        method: "POST",
      });
    } catch (error) {
      console.error("Error hanging up:", error);
    }
    serverScreenSharePeerId = null;
  }

  if (serverScreenSharePC) {
    serverScreenSharePC.close();
    serverScreenSharePC = null;
  }

  const videoElement = document.getElementById("screen-share-video");
  if (videoElement) {
    videoElement.srcObject = null;
  }
}

// Stop LiveKit screen sharing
async function stopLiveKitScreenShare() {
  if (screenTrack && screenTrack.length > 0) {
    console.log("Stopping LiveKit screen share");
    for (const track of screenTrack) {
      // Unpublish from LiveKit room
      if (room) {
        try {
          room.localParticipant.unpublishTrack(track);
        } catch (e) {
          console.warn("Error unpublishing track:", e);
        }
      }
      // Stop the underlying MediaStreamTrack
      if (track.mediaStreamTrack) {
        track.mediaStreamTrack.stop();
      }
      // Stop the LiveKit track
      track.stop();
    }
    screenTrack = null;
  }

  // Clear the video element to remove the black rectangle
  const videoElement = document.getElementById("screen-share-video");
  if (videoElement) {
    videoElement.srcObject = null;
  }

  // Reset button state
  btnRoomScreen.classList.remove("active");
  btnRoomScreen.classList.remove("circle-btn--screen-share-on");
}

/**
 * Unified function to stop all types of screen sharing and reset UI state.
 * This ensures that whether sharing was started via server or LiveKit,
 * and whether it was stopped manually or via browser UI, the state is fully reset.
 */
async function stopAllScreenSharing() {
  console.log("Executing unified screen share cleanup");
  // Stop both potential sharing methods
  await stopServerScreenShare();
  await stopLiveKitScreenShare();

  // Reset global state
  isScreenSharing = false;

  // Reset UI view
  setRoomView("default");
  console.log("Screen sharing state fully reset");
}

// LiveKit screen sharing (fallback)
async function startLiveKitScreenShare() {
  if (!room) {
    throw new Error("Room not initialized");
  }

  const { createLocalScreenTracks } = window.LiveKitClient || {};
  if (!createLocalScreenTracks) {
    throw new Error("LiveKit functions not available");
  }

  screenTrack = await createLocalScreenTracks({
    resolution: { width: 1920, height: 1080 },
  });
  if (screenTrack && screenTrack.length > 0) {
    const track = screenTrack[0];
    await room.localParticipant.publishTrack(track);
    attachVideoTrack(track, "screen-share-video");

    // Listen for the browser's native "Stop sharing" button
    // This fires when user clicks the browser UI to stop sharing
    const mediaStreamTrack = track.mediaStreamTrack;
    if (mediaStreamTrack) {
      mediaStreamTrack.addEventListener('ended', async () => {
        console.log("Screen share stopped via browser UI (LiveKit)");
        await stopAllScreenSharing();
      });
    }
  }
}

// Чат
btnRoomChat.addEventListener("click", () => {
  if (currentRoomView === "chat") {
    setRoomView("default");
  } else {
    setRoomView("chat");
  }
});

chatSend.addEventListener("click", () => {
  sendChatMessage();
});

chatInput.addEventListener("keydown", (e) => {
  if (e.key === "Enter" && !e.shiftKey) {
    e.preventDefault();
    sendChatMessage();
  }
});

// Список участников
btnRoomParticipants.addEventListener("click", () => {
  showParticipantsModal();
});

participantsClose.addEventListener("click", () => {
  participantsModal.classList.add("hidden");
});

async function showParticipantsModal() {
  participantsModal.classList.remove("hidden");
  participantsList.innerHTML = '<div class="loading-spinner">Загрузка...</div>';
  participantsCount.textContent = "Загрузка...";

  try {
    const participants = await loadParticipants();
    renderParticipants(participants);
  } catch (error) {
    console.error("Error loading participants:", error);
    participantsList.innerHTML = `<div class="error-text">Ошибка: ${error.message}</div>`;
    participantsCount.textContent = "Ошибка загрузки";
  }
}

async function loadParticipants() {
  if (!currentRoomId) throw new Error("Room ID not found");

  const response = await fetch(`${API_BASE}/rooms/${currentRoomId}/participants`);
  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error || "Failed to fetch participants");
  }

  return await response.json();
}

function renderParticipants(participants) {
  participantsList.innerHTML = "";
  participantsCount.textContent = `Всего участников: ${participants.length}`;

  // Получаем ID текущего участника из localStorage или другого места
  const myParticipantId = localStorage.getItem("participant_id");

  participants.forEach(p => {
    const item = document.createElement("div");
    item.className = "participant-item";

    const initials = p.display_name ? p.display_name.charAt(0).toUpperCase() : "?";
    const isMe = p.id === myParticipantId;
    const isHost = p.role === "host";

    let badges = "";
    if (isHost) badges += '<span class="participant-badge participant-badge--host">Организатор</span>';
    if (isMe) badges += '<span class="participant-badge participant-badge--me">Вы</span>';

    item.innerHTML = `
      <div class="participant-info">
        <div class="participant-name">${p.display_name || "Аноним"}</div>
      </div>
      <div class="participant-badges">
        ${badges}
      </div>
    `;

    participantsList.appendChild(item);
  });
}

async function sendChatMessage() {
  const text = chatInput.value.trim();
  if (!text || !currentRoomId) return;

  // Получаем display_name
  const displayName = localStorage.getItem("display_name") || "User";

  // Отправка через API
  try {
    const response = await fetch(`${API_BASE}/rooms/${currentRoomId}/chat/messages`, {
      method: "POST",
      headers: getAPIHeaders(),
      body: JSON.stringify({ content: text, display_name: displayName }),
    });

    if (response.ok) {
      appendChatMessage({ author: "Я", text, me: true });
      chatInput.value = "";
    } else {
      console.error("Chat message failed:", response.status, await response.text());
    }
  } catch (error) {
    console.error("Error sending chat message:", error);
  }

  // Также отправляем через LiveKit data channel для реального времени
  if (room) {
    try {
      const { DataPacket_Kind } = window.LiveKitClient || {};
      const data = JSON.stringify({ type: "chat", message: text });
      room.localParticipant.publishData(
        new TextEncoder().encode(data),
        DataPacket_Kind?.RELIABLE || 0
      );
    } catch (error) {
      console.error("Error sending via LiveKit:", error);
    }
  }
}

function appendChatMessage({ author, text, me }) {
  const div = document.createElement("div");
  div.className = `chat-message ${me ? "chat-message--me" : "chat-message--guest"}`;
  div.innerHTML = `<div class="chat-author">${author}</div><div>${text}</div>`;
  chatMessages.appendChild(div);
  chatMessages.scrollTop = chatMessages.scrollHeight;
}

// Показать отдельное окно с ID комнаты
function showRoomIdModal(roomId) {
  const backdrop = document.createElement("div");
  backdrop.className = "modal-backdrop";
  backdrop.innerHTML = `
    <div class="modal">
      <div class="modal-title">ID вашей комнаты</div>
      <div class="modal-sub">Поделитесь этим ID, чтобы участники могли подключиться</div>
      <input class="modal-input" value="${roomId}" readonly />
      <div class="modal-actions">
        <button class="modal-btn modal-btn--primary" id="room-id-ok">Понятно</button>
      </div>
    </div>
  `;
  document.body.appendChild(backdrop);

  document.getElementById("room-id-ok").addEventListener("click", () => {
    backdrop.remove();
  });
}

// Завершение звонка
btnRoomLeave.addEventListener("click", async () => {
  if (room) {
    try {
      if (localVideoTrack) {
        localVideoTrack.stop();
      }
      if (localAudioTrack) {
        localAudioTrack.stop();
      }
      if (screenTrack) {
        if (Array.isArray(screenTrack)) {
          screenTrack.forEach((t) => t.stop());
        } else {
          screenTrack.stop();
        }
      }
      await stopServerScreenShare();
      await room.disconnect();
    } catch (error) {
      console.error("Error disconnecting:", error);
    }
    room = null;
  }

  // Уведомление сервера о выходе
  if (currentRoomId) {
    try {
      await fetch(`${API_BASE}/rooms/${currentRoomId}/leave`, {
        method: "POST",
        headers: getAPIHeaders(),
      });
    } catch (error) {
      console.error("Error leaving room:", error);
    }
  }

  // Останавливаем определение активного говорящего
  stopActiveSpeakerDetection();
  
  // Очищаем анализаторы аудио
  audioAnalysers.forEach((analyserData, identity) => {
    if (analyserData.audioContext) {
      analyserData.audioContext.close().catch(console.warn);
    }
  });
  audioAnalysers.clear();

  // Очищаем сохранённую комнату при явном выходе
  clearSavedRoomId();

  window.location.href = "/";
});

// Функция ожидания загрузки LiveKit SDK
async function waitForLiveKit(maxAttempts = 100, interval = 100) {
  // Используем глобальный Promise из room.html, если доступен
  if (window.liveKitLoadPromise) {
    try {
      console.log("Ожидание загрузки LiveKit SDK через глобальный Promise...");
      await window.liveKitLoadPromise;

      // После успешной загрузки ищем SDK
      const sdk = getLiveKitSDK();
      if (sdk && sdk.Room) {
        console.log("LiveKit SDK готов (через глобальный Promise):", Object.keys(sdk).slice(0, 10));
        return sdk;
      }
    } catch (err) {
      console.warn("Ошибка глобального Promise загрузки LiveKit:", err);
    }
  }

  // Fallback: если глобальный Promise не сработал, используем polling
  return new Promise(async (resolve, reject) => {
    // Пробуем загрузить через функцию, если доступна
    if (window.loadLiveKitScript && typeof window.loadLiveKitScript === 'function') {
      try {
        await window.loadLiveKitScript();
      } catch (err) {
        console.warn("Ошибка при загрузке LiveKit через loadLiveKitScript:", err);
      }
    }

    let attempts = 0;
    const checkLiveKit = () => {
      attempts++;
      const LiveKit = getLiveKitSDK();

      if (LiveKit && LiveKit.Room) {
        console.log("LiveKit SDK найден:", LiveKit);
        console.log("Доступные методы:", Object.keys(LiveKit).slice(0, 20));
        resolve(LiveKit);
        return;
      }

      if (attempts >= maxAttempts) {
        // Детальная диагностика
        console.error("=== ДЕТАЛЬНАЯ ДИАГНОСТИКА LiveKit SDK ===");
        console.error("Попыток проверки:", attempts);

        const allGlobals = Object.keys(window).filter(k =>
          k.toLowerCase().includes('livekit') ||
          k.toLowerCase().includes('room') ||
          k.toLowerCase().includes('track')
        );
        console.error("Глобальные переменные с 'livekit', 'room', 'track':", allGlobals);

        reject(new Error("LiveKit SDK не загрузился за отведенное время. Проверьте консоль браузера для деталей."));
        return;
      }

      setTimeout(checkLiveKit, interval);
    };

    // Начинаем проверку сразу
    checkLiveKit();
  });
}

// Обработчики для модального окна выбора устройств
if (btnRoomSettings) {
  btnRoomSettings.addEventListener("click", () => {
    showDeviceSettingsModal();
  });
}

if (deviceSettingsCancel) {
  deviceSettingsCancel.addEventListener("click", () => {
    hideDeviceSettingsModal();
  });
}

if (deviceSettingsApply) {
  deviceSettingsApply.addEventListener("click", () => {
    applyDeviceSettings();
  });
}

// Закрытие модального окна по клику вне его
if (deviceSettingsModal) {
  deviceSettingsModal.addEventListener("click", (e) => {
    if (e.target === deviceSettingsModal) {
      hideDeviceSettingsModal();
    }
  });
}

// Копирование ID комнаты
if (copyRoomIdBtn) {
  copyRoomIdBtn.addEventListener("click", async () => {
    if (!currentRoomId) {
      alert("ID комнаты еще не загружен");
      return;
    }

    // Construct the full URL
    const roomUrl = window.location.origin + window.location.pathname + '?room=' + currentRoomId;

    try {
      await navigator.clipboard.writeText(roomUrl);

      // Визуальная обратная связь
      copyRoomIdBtn.classList.add("copied");
      const originalTitle = copyRoomIdBtn.getAttribute("title");
      copyRoomIdBtn.setAttribute("title", "Ссылка скопирована!");

      setTimeout(() => {
        copyRoomIdBtn.classList.remove("copied");
        copyRoomIdBtn.setAttribute("title", originalTitle || "Копировать ссылку на комнату");
      }, 2000);
    } catch (error) {
      console.error("Ошибка копирования:", error);
      // Fallback для старых браузеров
      const textArea = document.createElement("textarea");
      textArea.value = roomUrl;
      textArea.style.position = "fixed";
      textArea.style.opacity = "0";
      document.body.appendChild(textArea);
      textArea.select();
      try {
        document.execCommand("copy");
        copyRoomIdBtn.classList.add("copied");
        const originalTitle = copyRoomIdBtn.getAttribute("title");
        copyRoomIdBtn.setAttribute("title", "Ссылка скопирована!");
        setTimeout(() => {
          copyRoomIdBtn.classList.remove("copied");
          copyRoomIdBtn.setAttribute("title", originalTitle || "Копировать ссылку на комнату");
        }, 2000);
      } catch (fallbackError) {
        console.error("Ошибка fallback копирования:", fallbackError);
        alert("Не удалось скопировать ссылку. Ссылка: " + roomUrl);
      }
      document.body.removeChild(textArea);
    }
  });
}

// Инициализация при загрузке страницы
document.addEventListener("DOMContentLoaded", async () => {
  console.log("DOM загружен, начинаем инициализацию LiveKit...");

  try {
    // Ждем загрузки LiveKit SDK (максимум 10 секунд, проверяем каждые 100мс)
    console.log("Ожидание загрузки LiveKit SDK...");
    const LiveKit = await waitForLiveKit(100, 100);

    // Извлекаем нужные функции из SDK
    const Room = LiveKit.Room || (LiveKit.default && LiveKit.default.Room);
    const createLocalAudioTrack = LiveKit.createLocalAudioTrack || (LiveKit.default && LiveKit.default.createLocalAudioTrack);
    const createLocalVideoTrack = LiveKit.createLocalVideoTrack || (LiveKit.default && LiveKit.default.createLocalVideoTrack);
    const createLocalScreenTracks = LiveKit.createLocalScreenTracks || (LiveKit.default && LiveKit.default.createLocalScreenTracks);
    const DataPacket_Kind = LiveKit.DataPacket_Kind || (LiveKit.default && LiveKit.default.DataPacket_Kind);

    // Сохраняем в window для использования в других местах
    window.LiveKitClient = {
      Room,
      createLocalAudioTrack,
      createLocalVideoTrack,
      createLocalScreenTracks,
      DataPacket_Kind
    };

    console.log("LiveKit SDK готов к использованию");
    console.log("Доступные функции:", Object.keys(window.LiveKitClient));

    // Небольшая задержка для гарантии полной инициализации
    setTimeout(() => {
      console.log("Запуск initRoom...");
      initRoom();
    }, 300);
  } catch (error) {
    console.error("Критическая ошибка загрузки LiveKit SDK:", error);
    console.error("Стек ошибки:", error.stack);

    if (roomBanner) {
      roomBanner.textContent = "Ошибка: LiveKit SDK не загружен";
    }
    if (roomSubtitle) {
      roomSubtitle.textContent = error.message || "Проверьте консоль браузера";
    }

    // Показываем более информативное сообщение
    const errorMsg = `Не удалось загрузить LiveKit SDK.\n\n` +
      `Возможные причины:\n` +
      `1. Проблемы с интернет-соединением\n` +
      `2. Блокировка CDN файрволом или расширениями браузера\n` +
      `3. Проблемы с CDN серверами\n\n` +
      `Попробуйте:\n` +
      `- Обновить страницу (F5)\n` +
      `- Проверить консоль браузера (F12)\n` +
      `- Отключить блокировщики рекламы\n` +
      `- Проверить подключение к интернету`;

    alert(errorMsg);
  }
});
