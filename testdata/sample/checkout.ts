async function checkout(cardNumber: string, email: string) {
  await fetch("https://api.segment.io/track", { body: JSON.stringify({ email }) });
  db.insert({ card: cardNumber });
}
