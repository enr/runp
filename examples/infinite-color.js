#!/usr/bin/env node

/**
 * Script Node.js che stampa messaggi random con colori random
 * ogni 0.5-2 secondi, fino a SIGINT
 */

// Funzione per generare un numero random tra min e max
function randomBetween(min, max) {
  return Math.random() * (max - min) + min;
}

// Funzione per generare una stringa random di lunghezza specificata
function generateRandomString(length) {
  const characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()_+-=[]{}|;:,.<>?';
  let result = '';
  for (let i = 0; i < length; i++) {
      result += characters.charAt(Math.floor(Math.random() * characters.length));
  }
  return result;
}

// Codici ANSI per i colori del testo
const colors = {
  reset: '\x1b[0m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  magenta: '\x1b[35m',
  cyan: '\x1b[36m',
  white: '\x1b[37m',
  brightRed: '\x1b[91m',
  brightGreen: '\x1b[92m',
  brightYellow: '\x1b[93m',
  brightBlue: '\x1b[94m',
  brightMagenta: '\x1b[95m',
  brightCyan: '\x1b[96m'
};

// Array delle chiavi dei colori disponibili
const colorKeys = Object.keys(colors).filter(key => key !== 'reset');

// Funzione per ottenere un colore random
function getRandomColor() {
  return colors[colorKeys[Math.floor(Math.random() * colorKeys.length)]];
}

// Funzione per stampare un messaggio con colore random
function printRandomMessage() {
  const length = Math.floor(randomBetween(10, 50)); // Lunghezza tra 10 e 50 caratteri
  const message = generateRandomString(length);
  const color = getRandomColor();
  
  console.log(`${color}${message}${colors.reset}`);
}

// Gestione SIGINT (Ctrl+C)
process.on('SIGINT', () => {
  console.log('\n' + colors.yellow + 'Script interrotto dall\'utente. Arrivederci!' + colors.reset);
  process.exit(0);
});

// Funzione principale che esegue il loop
async function main() {
  console.log(colors.cyan + 'Script avviato. Premi Ctrl+C per interrompere.' + colors.reset);
  console.log(colors.cyan + 'Stampo messaggi random ogni 0.5-2 secondi...' + colors.reset + '\n');
  
  while (true) {
      const delay = randomBetween(500, 2000); // Delay tra 500ms e 2000ms
      printRandomMessage();
      await new Promise(resolve => setTimeout(resolve, delay));
  }
}

// Avvia lo script
main().catch(error => {
  console.error(colors.red + 'Errore:' + colors.reset, error);
  process.exit(1);
});