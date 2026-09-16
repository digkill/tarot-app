import Foundation

/// Turns the server's declarative card effects into sentences. Numbers come
/// from the catalog; wording from the translations.
enum ArcanaText {
    static func describe(_ side: ArcanaSide, _ l: Localizer) -> [String] {
        side.ops.compactMap { describe($0, l) }
    }

    static func describe(_ op: ArcanaOp, _ l: Localizer) -> String? {
        let amount = op.amount ?? 0
        let status = op.status.map { l.t("arcana.status.\($0)") } ?? ""
        switch op.kind {
        case "damage":
            if op.who == "self" { return l.t("arcana.op.damageSelf", ["amount": amount]) }
            var text = l.t(op.element == "magic" ? "arcana.op.damageMagic" : "arcana.op.damagePhysical", ["amount": amount])
            if op.pierce == true { text += ", " + l.t("arcana.op.pierce") }
            if let chance = op.chance, chance > 0 { text += " (" + l.t("arcana.op.crit", ["chance": chance]) + ")" }
            return text
        case "combo_damage":
            return l.t("arcana.op.combo", ["amount": amount, "bonus": op.bonus ?? 0, "status": status])
        case "lifesteal": return l.t("arcana.op.lifesteal", ["amount": amount])
        case "heal": return l.t("arcana.op.heal", ["amount": amount])
        case "armor": return l.t("arcana.op.armor", ["amount": amount])
        case "status":
            let key = op.who == "self" ? "arcana.op.statusSelf" : "arcana.op.statusTarget"
            let text = l.t(key, ["status": status, "amount": amount])
            if let turns = op.turns, turns > 0 {
                return l.t("arcana.op.statusTurns", ["text": text, "count": turns])
            }
            return text
        case "cleanse": return l.t("arcana.op.cleanse")
        case "draw": return l.t("arcana.op.draw", ["amount": amount])
        case "mana": return l.t("arcana.op.mana", ["amount": amount])
        case "discard_random": return l.t("arcana.op.discardRandom")
        case "redraw_hand":
            if let bonus = op.bonus, bonus > 0 { return l.t("arcana.op.redrawBonus", ["bonus": bonus]) }
            return l.t("arcana.op.redraw")
        case "repeat_last": return l.t(op.who == "opponent" ? "arcana.op.repeatOpponent" : "arcana.op.repeatSelf")
        case "purge": return l.t(op.bonus ?? 0 > 0 ? "arcana.op.purgeTarget" : "arcana.op.purgeBoth", ["amount": amount])
        case "wheel": return l.t("arcana.op.wheel")
        case "discount_hand": return l.t("arcana.op.discountHand", ["amount": amount])
        case "recall": return l.t("arcana.op.recall")
        case "recycle": return l.t("arcana.op.recycle")
        case "copy_card": return l.t("arcana.op.copyCard")
        case "reap": return l.t("arcana.op.reap", ["amount": amount])
        case "soul_harvest": return l.t("arcana.op.soulHarvest", ["amount": amount])
        default: return nil
        }
    }

    /// One line of the battle log, from the viewer's side.
    static func logLine(_ entry: ArcanaLogEntry, me: String, cardName: (String) -> String, _ l: Localizer) -> String? {
        let who = entry.player == me ? l.t("arcana.you") : l.t("arcana.opponent")
        switch entry.action {
        case "card.play":
            if let id = entry.cardId {
                return l.t("arcana.log.card", ["player": who, "card": cardName(id)])
            }
            return l.t("arcana.log.hiddenCard", ["player": who])
        case "hero.ability": return l.t("arcana.log.ability", ["player": who])
        case "hero.ultimate": return l.t("arcana.log.ultimate", ["player": who])
        case "turn.end": return l.t("arcana.log.endTurn", ["player": who])
        case "turn.timeout": return l.t("arcana.log.timeout", ["player": who])
        case "mulligan": return l.t("arcana.log.mulligan", ["player": who])
        case "surrender": return l.t("arcana.log.surrender", ["player": who])
        case "mulligan.timeout": return l.t("arcana.log.start")
        default: return nil
        }
    }
}
