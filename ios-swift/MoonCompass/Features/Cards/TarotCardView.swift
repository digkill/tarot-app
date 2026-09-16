import SwiftUI

/// A card that can start face down and flip over when tapped.
///
/// A reversed card is its art turned upside down.
struct TarotCardView: View {
    @Environment(SettingsStore.self) private var settings
    @Environment(\.appColors) private var colors

    let card: TarotCard
    let isReversed: Bool
    let width: CGFloat
    var startFaceDown = false
    var interactive = true
    /// Called when the face is tapped; after the flip if it was face down.
    var onOpen: () -> Void = {}

    @State private var rotation: Double?

    var body: some View {
        FlippingCard(
            rotation: rotation ?? (startFaceDown ? 0 : 180),
            faceFile: card.imageFile,
            isReversed: isReversed,
            width: width
        )
        .overlay(
            RoundedRectangle(cornerRadius: width * 0.08, style: .continuous)
                .stroke(colors.gold.opacity(0.35), lineWidth: 1)
        )
        .shadow(color: .black.opacity(0.35), radius: 6, y: 3)
        .contentShape(Rectangle())
        .onTapGesture(perform: tap)
        .allowsHitTesting(interactive)
        .accessibilityElement()
        .accessibilityLabel(card.name)
        .accessibilityAddTraits(.isButton)
    }

    private var isFaceUp: Bool {
        (rotation ?? (startFaceDown ? 0 : 180)) >= 90
    }

    private func tap() {
        guard !isFaceUp else {
            onOpen()
            return
        }
        guard !settings.settings.disableAnimations else {
            rotation = 180
            onOpen()
            return
        }
        withAnimation(.easeOut(duration: 0.6)) {
            rotation = 180
        } completion: {
            onOpen()
        }
    }
}

/// Draws whichever side faces the viewer at the *animated* angle.
///
/// Being `Animatable`, SwiftUI re-renders it with every intermediate rotation,
/// so the sides swap exactly at 90°. Deciding from the state value instead
/// would jump to the face the moment the flip starts.
private struct FlippingCard: View, Animatable {
    var rotation: Double
    let faceFile: String
    let isReversed: Bool
    let width: CGFloat

    nonisolated var animatableData: Double {
        get { rotation }
        set { rotation = newValue }
    }

    var body: some View {
        ZStack {
            if rotation < 90 {
                CardImage(file: CardArt.backFile, width: width)
            } else {
                // Pre-turned by 180° so it reads correctly once flipped over.
                CardImage(file: faceFile, width: width)
                    .rotationEffect(.degrees(isReversed ? 180 : 0))
                    .rotation3DEffect(.degrees(180), axis: (x: 0, y: 1, z: 0))
            }
        }
        .frame(width: width, height: width * CardMetrics.aspect)
        .rotation3DEffect(.degrees(rotation), axis: (x: 0, y: 1, z: 0), perspective: 0.5)
    }
}
