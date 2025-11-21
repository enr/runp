#!/usr/bin/env node

/**
 * Node.js script that prints random messages with random colors
 * every 0.5-2 seconds, until SIGINT
 */

// Function to generate a random number between min and max
function randomBetween(min, max) {
  return Math.random() * (max - min) + min;
}

// Function to generate a random string of specified length
function generateRandomString(length) {
  const characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()_+-=[]{}|;:,.<>?';
  let result = '';
  for (let i = 0; i < length; i++) {
      result += characters.charAt(Math.floor(Math.random() * characters.length));
  }
  return result;
}

// ANSI color codes for text
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

// Array of available color keys
const colorKeys = Object.keys(colors).filter(key => key !== 'reset');

// Function to get a random color
function getRandomColor() {
  return colors[colorKeys[Math.floor(Math.random() * colorKeys.length)]];
}

// Function to print a message with random color
function printRandomMessage() {
  const length = Math.floor(randomBetween(10, 50)); // Length between 10 and 50 characters
  const message = generateRandomString(length);
  const color = getRandomColor();
  
  console.log(`${color}${message}${colors.reset}`);
}

// SIGINT handler (Ctrl+C)
process.on('SIGINT', () => {
  console.log('\n' + colors.yellow + 'Script terminated by user. Exiting...' + colors.reset);
  process.exit(0);
});

// Main function that executes the loop
async function main() {
  console.log(colors.cyan + 'Script started. Press Ctrl+C to terminate.' + colors.reset);
  console.log(colors.cyan + 'Printing random messages every 0.5-2 seconds...' + colors.reset + '\n');
  
  while (true) {
      const delay = randomBetween(500, 2000); // Delay between 500ms and 2000ms
      printRandomMessage();
      await new Promise(resolve => setTimeout(resolve, delay));
  }
}

// Start the script
main().catch(error => {
  console.error(colors.red + 'Error:' + colors.reset, error);
  process.exit(1);
});