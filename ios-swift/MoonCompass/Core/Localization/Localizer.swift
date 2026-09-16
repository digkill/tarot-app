import Foundation

/// Looks up UI strings in the i18next JSON files shared with the React Native
/// client (`i18n/<lang>.json` plus `i18n/legal_<lang>.json`), with iOS-only
/// strings from `i18n_ios/<lang>.json` layered on top.
///
/// Apple's string catalogs are not used on purpose: the app lets the user pick
/// a language in its own settings, independently of the device language, and
/// string catalogs follow the device. Reading the shared JSON also keeps a
/// single source of truth for both apps.
struct Localizer: Sendable {
    let language: Language

    private let strings: [String: String]
    private let lists: [String: [String]]
    private let fallbackStrings: [String: String]
    private let fallbackLists: [String: [String]]

    init(language: Language, bundle: Bundle = .main) {
        self.language = language
        let own = Self.load(language, bundle: bundle)
        strings = own.strings
        lists = own.lists
        if language == .en {
            fallbackStrings = [:]
            fallbackLists = [:]
        } else {
            let english = Self.load(.en, bundle: bundle)
            fallbackStrings = english.strings
            fallbackLists = english.lists
        }
    }

    /// Builds a localizer from in-memory tables — for tests and previews.
    init(language: Language, strings: [String: String], lists: [String: [String]] = [:],
         fallbackStrings: [String: String] = [:], fallbackLists: [String: [String]] = [:]) {
        self.language = language
        self.strings = strings
        self.lists = lists
        self.fallbackStrings = fallbackStrings
        self.fallbackLists = fallbackLists
    }

    /// The translated string for a dotted key, with `{{name}}` placeholders
    /// filled from `arguments`. Falls back to English, then to the key itself —
    /// the same visible failure i18next produces, so a missing key is obvious.
    func t(_ key: String, _ arguments: [String: CustomStringConvertible] = [:]) -> String {
        let resolvedKey = pluralKey(for: key, arguments: arguments)
        let template = strings[resolvedKey]
            ?? fallbackStrings[resolvedKey]
            ?? strings[key]
            ?? fallbackStrings[key]
            ?? key
        return Self.interpolate(template, arguments)
    }

    /// Whether a translation exists — used to map server error codes onto
    /// `auth.errors.<code>` only when that message was actually written.
    func has(_ key: String) -> Bool {
        strings[key] != nil || fallbackStrings[key] != nil
    }

    /// A translated list, such as the paragraphs of a legal document.
    func list(_ key: String) -> [String] {
        lists[key] ?? fallbackLists[key] ?? []
    }

    // MARK: - Plurals

    /// i18next resolves `key` with a `count` argument to `key_one`, `key_few`,
    /// `key_many` or `key_other` using the language's plural rules.
    private func pluralKey(for key: String, arguments: [String: CustomStringConvertible]) -> String {
        guard let count = arguments["count"].flatMap({ Int(String(describing: $0)) }) else {
            return key
        }
        let candidate = "\(key)_\(PluralRules.category(for: count, in: language).rawValue)"
        if strings[candidate] != nil || fallbackStrings[candidate] != nil {
            return candidate
        }
        let other = "\(key)_other"
        if strings[other] != nil || fallbackStrings[other] != nil {
            return other
        }
        return key
    }

    // MARK: - Loading

    private struct Tables {
        var strings: [String: String] = [:]
        var lists: [String: [String]] = [:]
    }

    private static func load(_ language: Language, bundle: Bundle) -> Tables {
        var tables = Tables()
        // The iOS overlay comes last, so its keys win over the shared ones.
        let files = [("i18n", language.rawValue), ("i18n", "legal_\(language.rawValue)"), ("i18n_ios", language.rawValue)]
        for (folder, name) in files {
            guard let url = bundle.url(forResource: name, withExtension: "json", subdirectory: folder),
                  let data = try? Data(contentsOf: url),
                  let object = try? JSONSerialization.jsonObject(with: data) else {
                continue
            }
            flatten(object, prefix: "", into: &tables)
        }
        return tables
    }

    private static func flatten(_ object: Any, prefix: String, into tables: inout Tables) {
        switch object {
        case let dictionary as [String: Any]:
            for (key, value) in dictionary {
                flatten(value, prefix: prefix.isEmpty ? key : "\(prefix).\(key)", into: &tables)
            }
        case let array as [Any]:
            tables.lists[prefix] = array.compactMap { $0 as? String }
        case let string as String:
            tables.strings[prefix] = string
        case let number as NSNumber:
            tables.strings[prefix] = number.stringValue
        default:
            break
        }
    }

    static func interpolate(_ template: String, _ arguments: [String: CustomStringConvertible]) -> String {
        guard !arguments.isEmpty, template.contains("{{") else { return template }
        var result = template
        for (name, value) in arguments {
            result = result.replacingOccurrences(of: "{{\(name)}}", with: String(describing: value))
            result = result.replacingOccurrences(of: "{{ \(name) }}", with: String(describing: value))
        }
        return result
    }
}

/// CLDR cardinal plural categories for the four supported languages.
enum PluralRules {
    enum Category: String {
        case one, few, many, other
    }

    static func category(for count: Int, in language: Language) -> Category {
        let n = abs(count)
        switch language {
        case .en:
            return n == 1 ? .one : .other
        case .ru:
            let mod10 = n % 10
            let mod100 = n % 100
            if mod10 == 1 && mod100 != 11 { return .one }
            if (2...4).contains(mod10) && !(12...14).contains(mod100) { return .few }
            return .many
        case .th, .zh:
            // Thai and Chinese do not inflect for number.
            return .other
        }
    }
}
