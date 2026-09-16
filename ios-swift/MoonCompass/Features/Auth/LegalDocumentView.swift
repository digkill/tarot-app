import SwiftUI

struct LegalDocumentView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(\.appColors) private var colors

    let document: LegalDocument

    var body: some View {
        let l = settings.localizer
        let paragraphs = l.list(document == .privacy ? "legal.privacyParagraphs" : "legal.termsParagraphs")
        ScrollView {
            VStack(alignment: .leading, spacing: 14) {
                Text(l.t("legal.version", ["version": SessionStore.consentVersion]))
                    .font(.footnote)
                    .foregroundStyle(colors.muted)
                ForEach(Array(paragraphs.enumerated()), id: \.offset) { _, paragraph in
                    Text(paragraph)
                        .font(.system(size: 15))
                        .lineSpacing(4)
                        .foregroundStyle(colors.text)
                        .wrapsText()
                }
            }
            .padding(20)
        }
        .navigationTitle(l.t(document == .privacy ? "legal.privacyTitle" : "legal.termsTitle"))
        .navigationBarTitleDisplayMode(.inline)
        .toolbarBackground(colors.bg, for: .navigationBar)
        .toolbarColorScheme(.dark, for: .navigationBar)
    }
}
