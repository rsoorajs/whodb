/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import {type SupportedLanguage, DEFAULT_LANGUAGE} from '@/utils/languages';

type TranslationMap = Record<string, string>;
type TranslationCache = Record<string, TranslationMap>;
type ParsedYaml = Record<string, TranslationMap>;

const translationCache: TranslationCache = {};

// Import all YAML files using Vite's import.meta.glob
// The yamlPlugin in vite.config.ts transforms these to pre-parsed JSON at build time.
const ceModules = import.meta.glob<ParsedYaml>('/src/locales/**/*.yaml', { import: 'default', eager: true });

/** Build a component-path → module index from raw glob results. */
const buildIndex = (modules: Record<string, ParsedYaml>): Map<string, ParsedYaml> => {
    const index = new Map<string, ParsedYaml>();
    for (const [key, mod] of Object.entries(modules)) {
        const match = key.match(/\/locales\/(.+)\.yaml$/);
        if (match) index.set(match[1], mod);
    }
    return index;
};

const ceIndex = buildIndex(ceModules);

// Extension locale modules — populated by registerLocaleModules()
let extensionIndex = new Map<string, ParsedYaml>();

/** Register additional locale modules (called by extensions at boot). */
export const registerLocaleModules = (modules: Record<string, ParsedYaml>) => {
    extensionIndex = buildIndex(modules);
};

/**
 * Cached Intl.PluralRules instances per locale.
 * Constructing PluralRules loads CLDR data internally — one instance per locale is sufficient.
 */
const pluralRulesCache = new Map<string, Intl.PluralRules>();

/**
 * Returns the CLDR plural category for a given count and locale.
 * Categories: "zero" | "one" | "two" | "few" | "many" | "other"
 */
export const getPluralCategory = (locale: string, count: number): Intl.LDMLPluralRule => {
    let rules = pluralRulesCache.get(locale);
    if (!rules) {
        rules = new Intl.PluralRules(locale.replace('_', '-'));
        pluralRulesCache.set(locale, rules);
    }
    return rules.select(count);
};

export const loadTranslationsSync = (
    componentPath: string,
    language: SupportedLanguage
): TranslationMap => {
    const cacheKey = `${componentPath}-${language}`;

    if (translationCache[cacheKey]) {
        return translationCache[cacheKey];
    }

    try {
        let translations: TranslationMap | undefined;

        // Load CE locale files as the base
        const ceParsed = ceIndex.get(componentPath);
        if (ceParsed) {
            translations = ceParsed[language] || ceParsed[DEFAULT_LANGUAGE];
        }

        // Merge extension translations on top (overrides CE keys if present)
        const extParsed = extensionIndex.get(componentPath);
        if (extParsed) {
            const extTranslations = extParsed[language] || extParsed[DEFAULT_LANGUAGE];
            if (extTranslations) {
                translations = { ...translations, ...extTranslations };
            }
        }

        if (!translations) {
            console.error(`Translation file not found for ${componentPath}`, {
                availableCE: [...ceIndex.keys()],
                language
            });
            return {};
        }

        translationCache[cacheKey] = translations;
        return translations;
    } catch (error) {
        console.error(`Failed to load translations for ${componentPath}:`, error);
        return {};
    }
};

/**
 * Resolves a translation key, handling pluralization and interpolation.
 *
 * Pluralization: when `params` contains a numeric `count` property, the function
 * looks up `key_<category>` (e.g. `key_one`, `key_other`) using Intl.PluralRules
 * for the given locale. Falls back to the base `key` if no plural form is found.
 *
 * Interpolation: `{placeholder}` tokens in the template are replaced with
 * corresponding values from `params`.
 */
export const getTranslation = (
    translations: TranslationMap,
    key: string,
    language: SupportedLanguage,
    params?: Record<string, any>
): string => {
    let resolvedKey = key;

    // Pluralization: if params has a numeric `count`, resolve the plural form
    if (params != null && typeof params.count === 'number') {
        const category = getPluralCategory(language, params.count);
        const pluralKey = `${key}_${category}`;
        if (translations[pluralKey]) {
            resolvedKey = pluralKey;
        }
    }

    const template = translations[resolvedKey] || resolvedKey;

    // Skip regex when the template has no placeholders
    if (params != null && template.includes('{')) {
        return template.replace(/\{(\w+)\}/g, (match, paramKey) => {
            return String(params[paramKey] ?? match);
        });
    }

    return template;
};
