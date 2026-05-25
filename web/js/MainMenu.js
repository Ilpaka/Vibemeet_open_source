// Главный экран - упрощённый вариант для анонимных комнат
const btnCreateRoom = document.getElementById("btn-create-room");
const btnJoinRoom = document.getElementById("btn-join-room");
const roomIdInput = document.getElementById("room-id-input");

const API_BASE = window.location.origin + "/api/v1";

// Генерация UUID (с fallback для браузеров без crypto.randomUUID)
function generateUUID() {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    try {
      return crypto.randomUUID();
    } catch (e) {}
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
    const r = Math.random() * 16 | 0;
    const v = c === 'x' ? r : (r & 0x3 | 0x8);
    return v.toString(16);
  });
}

// Генерация и получение participant_id для анонимных пользователей
function getParticipantID() {
  let participantID = localStorage.getItem("participant_id");
  if (!participantID) {
    participantID = generateUUID();
    localStorage.setItem("participant_id", participantID);
  }
  return participantID;
}

// Получение display_name
function getDisplayName() {
  return localStorage.getItem("display_name") || "User";
}

// Показать модальное окно для ввода имени перед созданием/подключением
function showNameModal(callback) {
  const currentName = getDisplayName();
  
  const backdrop = document.createElement("div");
  backdrop.className = "modal-backdrop";
  backdrop.innerHTML = `
    <div class="modal">
      <div class="modal-title">Ваше имя</div>
      <div class="modal-sub">Как вас называть в комнате?</div>
      <input id="name-input" class="modal-input" type="text" placeholder="Введите ваше имя..." value="${currentName}" />
      <div class="modal-actions">
        <button class="modal-btn modal-btn--ghost" id="name-cancel">Отмена</button>
        <button class="modal-btn modal-btn--primary" id="name-confirm">Продолжить</button>
      </div>
    </div>
  `;
  document.body.appendChild(backdrop);

  const nameInput = document.getElementById("name-input");
  const cancelBtn = document.getElementById("name-cancel");
  const confirmBtn = document.getElementById("name-confirm");

  nameInput.focus();
  nameInput.select();

  cancelBtn.addEventListener("click", () => {
    backdrop.remove();
  });

  confirmBtn.addEventListener("click", () => {
    const name = nameInput.value.trim() || "User";
    localStorage.setItem("display_name", name);
    backdrop.remove();
    callback(name);
  });

  nameInput.addEventListener("keydown", (e) => {
    if (e.key === "Enter") {
      confirmBtn.click();
    }
  });
}

// Создание новой комнаты
if (btnCreateRoom) {
  btnCreateRoom.addEventListener("click", () => {
    showNameModal((name) => {
      // ВАЖНО: Очищаем старый room_id чтобы создалась НОВАЯ комната
      localStorage.removeItem("current_room_id");
      sessionStorage.removeItem("current_room_id");
      console.log("Cleared room_id, creating new room...");
      
      // Переходим на страницу комнаты без ID - она создаст новую
      window.location.href = "room.html";
    });
  });
}

// Присоединение по ID
if (btnJoinRoom) {
  btnJoinRoom.addEventListener("click", () => {
    const joinSection = document.querySelector(".join-room-section");
    
    // Если поле ввода скрыто, показываем его
    if (joinSection && joinSection.classList.contains("hidden")) {
      joinSection.classList.remove("hidden");
      // Фокусируемся на поле ввода после завершения анимации
      setTimeout(() => {
        if (roomIdInput) {
          roomIdInput.focus();
          roomIdInput.select();
        }
      }, 500);
      return;
    }
    
    // Если поле видно, проверяем ID и присоединяемся
    const roomId = roomIdInput ? roomIdInput.value.trim() : "";
    if (!roomId) {
      alert("Введите ID комнаты");
      if (roomIdInput) {
        roomIdInput.focus();
      }
      return;
    }
    
    showNameModal((name) => {
      window.location.href = `room.html?room=${encodeURIComponent(roomId)}`;
    });
  });
}

// Обработчик для кнопки "Присоединиться"
const btnJoinSubmit = document.getElementById("btn-join-submit");
if (btnJoinSubmit) {
  btnJoinSubmit.addEventListener("click", (e) => {
    e.preventDefault();
    e.stopPropagation();
    
    const roomId = roomIdInput ? roomIdInput.value.trim() : "";
    if (!roomId) {
      if (roomIdInput) {
        roomIdInput.focus();
        roomIdInput.style.borderColor = "#ff6b6b";
        setTimeout(() => {
          roomIdInput.style.borderColor = "";
        }, 2000);
      }
      return;
    }
    
    showNameModal((name) => {
      window.location.href = `room.html?room=${encodeURIComponent(roomId)}`;
    });
  });
}

// Обработка Enter в поле ввода ID
if (roomIdInput) {
  roomIdInput.addEventListener("keydown", (e) => {
    if (e.key === "Enter") {
      e.preventDefault();
      if (btnJoinSubmit) {
        btnJoinSubmit.click();
      }
    }
  });
}

// Старые обработчики для обратной совместимости (если страница использует старые кнопки)
const btnRegisterMain = document.getElementById("btn-register-main");
const btnLoginMain = document.getElementById("btn-login-main");

// Сохранение токенов в localStorage
function saveTokens(accessToken, refreshToken) {
  localStorage.setItem("accessToken", accessToken);
  localStorage.setItem("refreshToken", refreshToken);
}

function getAccessToken() {
  return localStorage.getItem("accessToken");
}

if (btnRegisterMain) {
  btnRegisterMain.addEventListener("click", () => {
    window.location.href = "CreateRoom.html";
  });
}

if (btnLoginMain) {
  btnLoginMain.addEventListener("click", async () => {
    const token = getAccessToken();
    if (token) {
      showRoomChoiceModal();
    } else {
      await showLoginModal();
    }
  });
}

function showLoginModal() {
  return new Promise((resolve) => {
    const backdrop = document.createElement("div");
    backdrop.className = "modal-backdrop";
    backdrop.innerHTML = `
      <div class="modal">
        <div class="modal-title">Вход в систему</div>
        <div class="modal-sub">Введите ваши данные для входа</div>
        <input id="login-email" class="modal-input" type="email" placeholder="Email" />
        <input id="login-password" class="modal-input" type="password" placeholder="Пароль" style="margin-top: 10px" />
        <div class="modal-actions">
          <button class="modal-btn modal-btn--ghost" id="login-cancel">Отмена</button>
          <button class="modal-btn modal-btn--primary" id="login-confirm">Войти</button>
        </div>
      </div>
    `;
    document.body.appendChild(backdrop);

    const emailInput = document.getElementById("login-email");
    const passwordInput = document.getElementById("login-password");
    const cancelBtn = document.getElementById("login-cancel");
    const confirmBtn = document.getElementById("login-confirm");

    emailInput.focus();

    cancelBtn.addEventListener("click", () => {
      backdrop.remove();
      resolve(false);
    });

    confirmBtn.addEventListener("click", async () => {
      const email = emailInput.value.trim();
      const password = passwordInput.value;

      if (!email || !password) {
        alert("Заполните все поля");
        return;
      }

      try {
        const response = await fetch(`${API_BASE}/auth/login`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ email, password }),
        });

        if (!response.ok) {
          const error = await response.json();
          alert("Ошибка входа: " + (error.error || "Неверные данные"));
          return;
        }

        const data = await response.json();
        saveTokens(data.access_token, data.refresh_token);
        backdrop.remove();
        resolve(true);
        showRoomChoiceModal();
      } catch (error) {
        console.error("Login error:", error);
        alert("Ошибка подключения к серверу");
      }
    });

    emailInput.addEventListener("keydown", (e) => {
      if (e.key === "Enter") {
        passwordInput.focus();
      }
    });

    passwordInput.addEventListener("keydown", (e) => {
      if (e.key === "Enter") {
        confirmBtn.click();
      }
    });
  });
}

// Модальное окно выбора: создать комнату или войти по ID
function showRoomChoiceModal() {
  const backdrop = document.createElement("div");
  backdrop.className = "modal-backdrop";
  backdrop.innerHTML = `
    <div class="modal">
      <div class="modal-title">Что вы хотите сделать?</div>
      <div class="modal-sub">Создайте новую комнату или введите ID существующей комнаты</div>
      <input id="room-choice-id" class="modal-input" placeholder="ID комнаты (UUID)" />
      <div class="modal-actions">
        <button class="modal-btn modal-btn--ghost" id="room-choice-create">Создать комнату</button>
        <button class="modal-btn modal-btn--primary" id="room-choice-join">Войти по ID</button>
      </div>
    </div>
  `;
  document.body.appendChild(backdrop);

  const input = document.getElementById("room-choice-id");
  const createBtn = document.getElementById("room-choice-create");
  const joinBtn = document.getElementById("room-choice-join");

  createBtn.addEventListener("click", () => {
    backdrop.remove();
    window.location.href = "room.html";
  });

  joinBtn.addEventListener("click", () => {
    const roomId = input.value.trim();
    if (!roomId) {
      alert("Введите ID комнаты");
      return;
    }
    backdrop.remove();
    window.location.href = `room.html?room=${encodeURIComponent(roomId)}`;
  });

  input.addEventListener("keydown", (e) => {
    if (e.key === "Enter") {
      joinBtn.click();
    }
  });
}

// FAQ аккордеон
document.addEventListener("DOMContentLoaded", () => {
  const faqQuestions = document.querySelectorAll(".faq-question");
  
  faqQuestions.forEach(question => {
    question.addEventListener("click", () => {
      const faqItem = question.closest(".faq-item");
      const isActive = faqItem.classList.contains("active");
      
      // Закрываем все остальные FAQ
      document.querySelectorAll(".faq-item").forEach(item => {
        if (item !== faqItem) {
          item.classList.remove("active");
        }
      });
      
      // Переключаем текущий FAQ
      faqItem.classList.toggle("active", !isActive);
    });
  });
  
  // Прокрутка к FAQ при клике на стрелку под hero-panel
  const heroScrollIndicator = document.querySelector(".hero-scroll-indicator");
  if (heroScrollIndicator) {
    heroScrollIndicator.addEventListener("click", () => {
      const faqSection = document.querySelector(".faq-section");
      if (faqSection) {
        faqSection.scrollIntoView({ behavior: "smooth", block: "start" });
      }
    });
  }
});