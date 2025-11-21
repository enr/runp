#!/usr/bin/env node

/**
 * Node.js script that deliberately prints problematic output
 * every 2 seconds, until SIGINT
 */

// Array of problematic messages
const problematicMessages = [
  // 1. Control characters
  () => "Message with\\nmultiple\\nnewlines\\n\\n",
  () => "Text with\\rcarriage return\\rthat overwrites",
  () => "Mix\\n\\r\\nof\\rcontrols",
  () => "TAB\\there\\tand\\there",
  
  // 2. Unclosed ANSI sequences
  () => "\\x1b[31mRed not reset",
  () => "\\x1b[1mBold\\x1b[32m then green",
  () => "\\x1b[44mBlue background\\x1b[5mBlinking",
  () => "\\x1b[7mInverted never reset",
  
  // 3. Problematic Unicode
  () => "Unicode characters: \\u0000\\u0001\\u0002\\u0003",
  () => "Extreme emoji: 🚀🔥💥" + "\\u200B\\u200C\\u200D", // zero-width chars
  () => "Bidi: Hello \\u202Eworld\\u202C", // right-to-left override
  () => "Unicode control characters: \\u001B\\u009B",
  
  // 4. Terminal special characters
  () => "\\x1b[2J\\x1b[HClear screen", // clear screen sequence
  () => "\\x1b[sSave cursor\\x1b[uRestore cursor",
  () => "\\x1b[?25lInvisible cursor\\x1b[?25h",
  
  // 5. Extreme lengths and formatting
  () => "A".repeat(200) + "\\n" + "B".repeat(150), // long lines
  () => "\\x1b[1000CExtreme right shift",
  () => "\\x1b[50AUpward shift",
  
  // 6. Mixed combinations
  () => "\\x1b[31mRed\\n\\x1b[32mGreen\\r\\x1b[33mYellow\\x1b[0m",
  () => "\\x1b[1;4;7mAll active" + "\\u202E" + "inverted text",
  () => "\\x1b]0;Window title\\x07Bell + title",
  
  // 7. Encoding issues
  () => Buffer.from([0xFF, 0xFE, 0xFD, 0xFC]).toString('binary'),
  () => "Half surrogate: \\uD800" + "text",
  
  // 8. Incomplete escape sequences
  () => "\\x1b[", // incomplete escape sequence
  () => "\\x1b[38;2;", // incomplete true color
  () => "\\x1b[#", // invalid parameter
  
  // 9. Invisible characters and spaces
  () => "Text\\u200Bwith\\u200Czero\\u200Dwidth",
  () => "\\u2800Braille\\u2800pattern\\u2800", // braille blank
  () => "\\u2066\\u2069Isolation", // directional isolates
  
  // 10. Partial resets and complex sequences
  () => "\\x1b[31m\\x1b[42mRed on green\\x1b[0mReset\\x1b[33mYellow remaining",
  () => "\\x1b[?1049hAlt buffer\\x1b[?1049l",
];

// Function to decode escape sequences
function decodeEscapeSequences(str) {
  return str
      .replace(/\\n/g, '\n')
      .replace(/\\r/g, '\r')
      .replace(/\\t/g, '\t')
      .replace(/\\x1b/g, '\x1b')
      .replace(/\\u([0-9a-fA-F]{4})/g, (_, hex) => String.fromCharCode(parseInt(hex, 16)))
      .replace(/\\\\/g, '\\');
}

// SIGINT handler
process.on('SIGINT', () => {
  // Before exiting, reset terminal state
  process.stdout.write('\x1b[0m\x1b[?25h\x1b[?1049l\n');
  console.log('\x1b[33mScript terminated. Terminal reset.\x1b[0m');
  process.exit(0);
});

// Initial information
console.log('\x1b[36m=== PROBLEMATIC OUTPUT GENERATOR ===\x1b[0m');
console.log('\x1b[33mThis script deliberately generates output that may:');
console.log('- Break terminal formatting');
console.log('- Leave unclosed ANSI sequences');
console.log('- Include control characters and problematic Unicode');
console.log('- Create visual artifacts');
console.log('Press Ctrl+C to terminate and reset the terminal\x1b[0m\n');

let counter = 0;

// Main function
async function main() {
  while (true) {
      const messageIndex = counter % problematicMessages.length;
      const messageGenerator = problematicMessages[messageIndex];
      const rawMessage = messageGenerator();
      const decodedMessage = decodeEscapeSequences(rawMessage);
      
      console.log(`\x1b[90m[${++counter}] \x1b[0m`);
      process.stdout.write(decodedMessage);
      process.stdout.write('\n'); // Force newline after each message
      
      await new Promise(resolve => setTimeout(resolve, 2000));
  }
}

// Start with error handling
main().catch(error => {
  process.stdout.write('\x1b[0m\x1b[?25h\n');
  console.error('\x1b[31mError:\x1b[0m', error);
  process.exit(1);
});