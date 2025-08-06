export async function generateTarotAnalysis(cards: string[]) {
    const res = await fetch("https://yourdomain.com/api/gpt-analysis", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({ cards }),
    });

    const data = await res.json();
    return data.analysis;
}
