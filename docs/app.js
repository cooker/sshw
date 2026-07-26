const i18n = window.SSHW_I18N;
const languageSelect = document.querySelector("#language-select");
const originalMessages = new Map();
const localeStorageKey = "sshw.pages.locale";
let currentLocale = i18n.defaultLocale;

function normalizeLocale(locale) {
  if (!locale) return null;
  const normalized = locale.toLowerCase();
  if (normalized === "zh" || normalized.startsWith("zh-")) return "zh-CN";
  return i18n.supportedLocales.find(item => item.toLowerCase() === normalized)
    || i18n.supportedLocales.find(item => normalized.startsWith(item.toLowerCase() + "-"))
    || null;
}

function storedLocale() {
  try {
    return localStorage.getItem(localeStorageKey);
  } catch (_) {
    return null;
  }
}

function rememberLocale(locale) {
  try {
    localStorage.setItem(localeStorageKey, locale);
  } catch (_) {
    // The page still works when storage is disabled.
  }
}

function cacheOriginalMessages() {
  const bindings = [
    ["[data-i18n]", "textContent", "data-i18n"],
    ["[data-i18n-html]", "innerHTML", "data-i18n-html"],
    ["[data-i18n-aria-label]", "aria-label", "data-i18n-aria-label"],
    ["[data-i18n-alt]", "alt", "data-i18n-alt"],
    ["[data-i18n-content]", "content", "data-i18n-content"]
  ];

  bindings.forEach(([selector, property, keyAttribute]) => {
    document.querySelectorAll(selector).forEach(element => {
      const key = element.getAttribute(keyAttribute);
      if (!originalMessages.has(key)) {
        originalMessages.set(key, property in element
          ? element[property]
          : element.getAttribute(property));
      }
    });
  });
}

function translate(key, locale = currentLocale) {
  const entry = i18n.translations[key];
  if (locale === i18n.defaultLocale && originalMessages.has(key)) {
    return originalMessages.get(key);
  }
  return entry?.[locale] ?? entry?.en ?? originalMessages.get(key) ?? key;
}

function applyLocale(locale, updateUrl = false) {
  currentLocale = normalizeLocale(locale) || i18n.defaultLocale;
  document.documentElement.lang = currentLocale;
  languageSelect.value = currentLocale;

  document.querySelectorAll("[data-i18n]").forEach(element => {
    element.textContent = translate(element.dataset.i18n);
  });
  document.querySelectorAll("[data-i18n-html]").forEach(element => {
    element.innerHTML = translate(element.dataset.i18nHtml);
  });
  document.querySelectorAll("[data-i18n-aria-label]").forEach(element => {
    element.setAttribute("aria-label", translate(element.dataset.i18nAriaLabel));
  });
  document.querySelectorAll("[data-i18n-alt]").forEach(element => {
    element.setAttribute("alt", translate(element.dataset.i18nAlt));
  });
  document.querySelectorAll("[data-i18n-content]").forEach(element => {
    element.setAttribute("content", translate(element.dataset.i18nContent));
  });

  rememberLocale(currentLocale);
  if (updateUrl) {
    const url = new URL(window.location.href);
    url.searchParams.set("lang", currentLocale);
    history.replaceState(null, "", url);
  }
}

cacheOriginalMessages();
const requestedLocale = new URLSearchParams(window.location.search).get("lang");
const initialLocale = normalizeLocale(requestedLocale)
  || normalizeLocale(storedLocale())
  || navigator.languages?.map(normalizeLocale).find(Boolean)
  || normalizeLocale(navigator.language)
  || i18n.defaultLocale;
applyLocale(initialLocale);

languageSelect.addEventListener("change", event => {
  applyLocale(event.target.value, true);
});

const menuButton = document.querySelector(".menu-button");
const siteNav = document.querySelector(".site-nav");
const toast = document.querySelector(".toast");
let toastTimer;

function closeMenu() {
  siteNav.classList.remove("open");
  menuButton.setAttribute("aria-expanded", "false");
}

menuButton.addEventListener("click", () => {
  const open = siteNav.classList.toggle("open");
  menuButton.setAttribute("aria-expanded", String(open));
});

siteNav.addEventListener("click", event => {
  if (event.target.closest("a")) closeMenu();
});

document.addEventListener("keydown", event => {
  if (event.key === "Escape") closeMenu();
});

function showToast(message) {
  toast.textContent = message;
  toast.classList.add("visible");
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => toast.classList.remove("visible"), 1800);
}

document.querySelectorAll("[data-copy]").forEach(button => {
  button.addEventListener("click", async () => {
    try {
      await navigator.clipboard.writeText(button.dataset.copy);
      showToast(translate("toast.copied"));
    } catch (_) {
      showToast(translate("toast.copyFailed"));
    }
  });
});

document.querySelectorAll("[data-tabs]").forEach(tabs => {
  const tabButtons = Array.from(tabs.querySelectorAll('[role="tab"]'));
  const panels = Array.from(tabs.querySelectorAll('[role="tabpanel"]'));

  function activateTab(button) {
    tabButtons.forEach(item => {
      const active = item === button;
      item.setAttribute("aria-selected", String(active));
      item.tabIndex = active ? 0 : -1;
    });
    panels.forEach(panel => {
      panel.hidden = panel.id !== button.getAttribute("aria-controls");
    });
  }

  tabButtons.forEach((button, index) => {
    button.addEventListener("click", () => activateTab(button));
    button.addEventListener("keydown", event => {
      if (!["ArrowLeft", "ArrowRight", "Home", "End"].includes(event.key)) return;
      event.preventDefault();
      let next = index;
      if (event.key === "ArrowRight") next = (index + 1) % tabButtons.length;
      if (event.key === "ArrowLeft") next = (index - 1 + tabButtons.length) % tabButtons.length;
      if (event.key === "Home") next = 0;
      if (event.key === "End") next = tabButtons.length - 1;
      activateTab(tabButtons[next]);
      tabButtons[next].focus();
    });
  });
});

const sectionLinks = Array.from(document.querySelectorAll('.site-nav a[href^="#"]'));
const observedSections = sectionLinks
  .map(link => document.querySelector(link.getAttribute("href")))
  .filter(Boolean);

if ("IntersectionObserver" in window) {
  const observer = new IntersectionObserver(entries => {
    const visible = entries
      .filter(entry => entry.isIntersecting)
      .sort((a, b) => b.intersectionRatio - a.intersectionRatio)[0];
    if (!visible) return;
    sectionLinks.forEach(link => {
      link.toggleAttribute("aria-current", link.getAttribute("href") === "#" + visible.target.id);
    });
  }, { rootMargin: "-20% 0px -65%", threshold: [0, .2, .5] });
  observedSections.forEach(section => observer.observe(section));
}
