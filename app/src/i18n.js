import i18n from "i18next";
import LanguageDetector from "i18next-browser-languagedetector";
import { initReactI18next } from "react-i18next";
import enTranslations from "./locales/en";
import zhTranslations from "./locales/zh";

const resources = {
  en: { translations: enTranslations },
  zh: { translations: zhTranslations }
};

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    react: {
      wait: true
    },
    resources: resources,
    lng: "en",
    fallbackLng: "en",
    debug: false,

    // have a common namespace used around the full app
    ns: ["translations"],
    defaultNS: "translations",

    keySeparator: ".",
    interpolation: {
      escapeValue: false
    },
    fallbackNS: []
  });

export default i18n;
