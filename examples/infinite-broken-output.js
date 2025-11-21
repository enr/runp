#!/usr/bin/env node

/**
 * Script Node.js che stampa deliberatamente output problematici
 * ogni 2 secondi, fino a SIGINT
 */

// Array di messaggi problematici
const problematicMessages = [
  // 1. Caratteri di controllo
  () => "Messaggio con\\nnewline\\ne multipli\\n\\n",
  () => "Testo con\\rcarriage return\\rche sovrascrive",
  () => "Mix\\n\\r\\n di\\r controlli",
  () => "TAB\\tqui\\te\\tqui",
  
  // 2. Sequenze ANSI non chiuse
  () => "\\x1b[31mRosso non resettato",
  () => "\\x1b[1mGrassetto\\x1b[32m poi verde",
  () => "\\x1b[44mSfondo blu\\x1b[5mBlinking",
  () => "\\x1b[7mInvertito mai resettato",
  
  // 3. Unicode problematico
  () => "Caratteri unicode: \\u0000\\u0001\\u0002\\u0003",
  () => "Emoji estreme: 🚀🔥💥" + "\\u200B\\u200C\\u200D", // zero-width chars
  () => "Bidi: Hello \\u202Eworld\\u202C", // right-to-left override
  () => "Caratteri di controllo unicode: \\u001B\\u009B",
  
  // 4. Caratteri speciali terminale
  () => "\\x1b[2J\\x1b[HClear screen", // sequenza clear screen
  () => "\\x1b[sSave cursor\\x1b[uRestore cursor",
  () => "\\x1b[?25lCursor invisibile\\x1b[?25h",
  
  // 5. Lunghezze estreme e formattazione
  () => "A".repeat(200) + "\\n" + "B".repeat(150), // linee lunghe
  () => "\\x1b[1000CSpostamento destro estremo",
  () => "\\x1b[50ASpostamento in alto",
  
  // 6. Combinazioni miste
  () => "\\x1b[31mRosso\\n\\x1b[32mVerde\\r\\x1b[33mGiallo\\x1b[0m",
  () => "\\x1b[1;4;7mTutto attivo" + "\\u202E" + "testo invertito",
  () => "\\x1b]0;Titolo finestra\\x07Bell + title",
  
  // 7. Problemi di encoding
  () => Buffer.from([0xFF, 0xFE, 0xFD, 0xFC]).toString('binary'),
  () => "Half surrogate: \\uD800" + "text",
  
  // 8. Sequenze di escape incomplete
  () => "\\x1b[", // escape sequence incompleta
  () => "\\x1b[38;2;", // true color incompleto
  () => "\\x1b[#", // parametro non valido
  
  // 9. Caratteri invisibili e spazi
  () => "Testo\\u200Bcon\\u200Czero\\u200Dwidth",
  () => "\\u2800Braille\\u2800pattern\\u2800", // braille blank
  () => "\\u2066\\u2069Isolamento", // directional isolates
  
  // 10. Reset parziali e sequenze complesse
  () => "\\x1b[31m\\x1b[42mRosso su verde\\x1b[0mReset\\x1b[33mGiallo rimasto",
  () => "\\x1b[?1049hAlt buffer\\x1b[?1049l",
];

// Funzione per decodificare le sequenze di escape
function decodeEscapeSequences(str) {
  return str
      .replace(/\\n/g, '\n')
      .replace(/\\r/g, '\r')
      .replace(/\\t/g, '\t')
      .replace(/\\x1b/g, '\x1b')
      .replace(/\\u([0-9a-fA-F]{4})/g, (_, hex) => String.fromCharCode(parseInt(hex, 16)))
      .replace(/\\\\/g, '\\');
}

// Gestione SIGINT
process.on('SIGINT', () => {
  // Prima di uscire, resettiamo lo stato del terminale
  process.stdout.write('\x1b[0m\x1b[?25h\x1b[?1049l\n');
  console.log('\x1b[33mScript interrotto. Terminale resettato.\x1b[0m');
  process.exit(0);
});

// Informazioni iniziali
console.log('\x1b[36m=== GENERATORE DI OUTPUT PROBLEMATICI ===\x1b[0m');
console.log('\x1b[33mQuesto script genera deliberatamente output che potrebbero:');
console.log('- Rompere la formattazione del terminale');
console.log('- Lasciare sequenze ANSI non chiuse');
console.log('- Includere caratteri di controllo e Unicode problematici');
console.log('- Creare artefatti visivi');
console.log('Premi Ctrl+C per interrompere e resettare il terminale\x1b[0m\n');

let counter = 0;

// Funzione principale
async function main() {
  while (true) {
      const messageIndex = counter % problematicMessages.length;
      const messageGenerator = problematicMessages[messageIndex];
      const rawMessage = messageGenerator();
      const decodedMessage = decodeEscapeSequences(rawMessage);
      
      console.log(`\x1b[90m[${++counter}] \x1b[0m`);
      process.stdout.write(decodedMessage);
      process.stdout.write('\n'); // Forziamo newline dopo ogni messaggio
      
      await new Promise(resolve => setTimeout(resolve, 2000));
  }
}

// Avvio con gestione errori
main().catch(error => {
  process.stdout.write('\x1b[0m\x1b[?25h\n');
  console.error('\x1b[31mErrore:\x1b[0m', error);
  process.exit(1);
});