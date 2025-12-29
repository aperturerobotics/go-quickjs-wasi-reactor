// QuickJS API Surface Test
console.log("=== QuickJS API Surface Test ===\n");

// Basic console output
console.log("1. Console output test");
console.log("  Hello from QuickJS!");

// Math operations
console.log("\n2. Math operations");
console.log("  2 + 2 =", 2 + 2);
console.log("  10 * 5 =", 10 * 5);
console.log("  Math.PI =", Math.PI);
console.log("  Math.sqrt(16) =", Math.sqrt(16));
console.log("  Math.max(1, 5, 3) =", Math.max(1, 5, 3));
console.log("  Math.random() =", Math.random());

// String operations
console.log("\n3. String operations");
console.log("  Concatenation:", "Hello" + " " + "World");
console.log("  Template literal:", `Result: ${2 + 2}`);
console.log("  toUpperCase:", "hello".toUpperCase());
console.log("  substring:", "JavaScript".substring(0, 4));
console.log("  split:", "a,b,c".split(","));

// Array operations
console.log("\n4. Array operations");
const arr = [1, 2, 3, 4, 5];
console.log("  Array:", arr);
console.log("  Length:", arr.length);
console.log("  map:", arr.map(x => x * 2));
console.log("  filter:", arr.filter(x => x > 2));
console.log("  reduce:", arr.reduce((a, b) => a + b, 0));
console.log("  find:", arr.find(x => x === 3));
console.log("  includes:", arr.includes(3));
console.log("  slice:", arr.slice(1, 3));

// Object operations
console.log("\n5. Object operations");
const obj = { name: "QuickJS", version: 1.0, active: true };
console.log("  Object:", obj);
console.log("  Keys:", Object.keys(obj));
console.log("  Values:", Object.values(obj));
console.log("  Entries:", Object.entries(obj));
console.log("  Access property:", obj.name);

// Functions
console.log("\n6. Function tests");
function add(a, b) {
  return a + b;
}
console.log("  Function result:", add(3, 4));

const multiply = (a, b) => a * b;
console.log("  Arrow function:", multiply(6, 7));

// Higher-order functions
const numbers = [1, 2, 3];
const doubled = numbers.map(n => n * 2);
console.log("  Higher-order function:", doubled);

// JSON operations
console.log("\n7. JSON operations");
const jsonObj = { foo: "bar", num: 42 };
const jsonStr = JSON.stringify(jsonObj);
console.log("  JSON.stringify:", jsonStr);
console.log("  JSON.parse:", JSON.parse(jsonStr));

// RegExp
console.log("\n8. RegExp tests");
const regex = /quick/i;
console.log("  RegExp test:", regex.test("QuickJS"));
console.log("  String match:", "QuickJS".match(/[A-Z]/g));
console.log("  String replace:", "Hello World".replace(/World/, "QuickJS"));

// Error handling
console.log("\n9. Error handling");
try {
  throw new Error("Test error");
} catch (e) {
  console.log("  Caught error:", e.message);
}

// Type checking
console.log("\n10. Type checking");
console.log("  typeof 42:", typeof 42);
console.log("  typeof 'hello':", typeof "hello");
console.log("  typeof true:", typeof true);
console.log("  typeof undefined:", typeof undefined);
console.log("  typeof null:", typeof null);
console.log("  typeof {}:", typeof {});
console.log("  typeof []:", typeof []);
console.log("  Array.isArray([]):", Array.isArray([]));

// Control flow
console.log("\n11. Control flow");
for (let i = 0; i < 3; i++) {
  console.log("  Loop iteration:", i);
}

let count = 0;
while (count < 3) {
  console.log("  While loop:", count);
  count++;
}

// Conditionals
console.log("\n12. Conditionals");
const value = 10;
if (value > 5) {
  console.log("  Value is greater than 5");
} else {
  console.log("  Value is 5 or less");
}

const result = value > 5 ? "greater" : "less or equal";
console.log("  Ternary result:", result);

// Destructuring
console.log("\n13. Destructuring");
const [a, b, c] = [1, 2, 3];
console.log("  Array destructuring:", a, b, c);

const { name, version } = obj;
console.log("  Object destructuring:", name, version);

// Spread operator
console.log("\n14. Spread operator");
const arr1 = [1, 2];
const arr2 = [3, 4];
const combined = [...arr1, ...arr2];
console.log("  Array spread:", combined);

const obj1 = { a: 1 };
const obj2 = { b: 2 };
const merged = { ...obj1, ...obj2 };
console.log("  Object spread:", merged);

// Rest parameters
console.log("\n15. Rest parameters");
function sum(...args) {
  return args.reduce((a, b) => a + b, 0);
}
console.log("  Rest parameters sum:", sum(1, 2, 3, 4, 5));

// Promise (if supported)
console.log("\n16. Promise test");
try {
  const promise = new Promise((resolve) => {
    resolve("Promise resolved!");
  });
  promise.then(result => console.log("  Promise result:", result));
} catch (e) {
  console.log("  Promises not supported or error:", e.message);
}

// Set and Map
console.log("\n17. Set and Map");
try {
  const set = new Set([1, 2, 3, 2, 1]);
  console.log("  Set:", Array.from(set));
  console.log("  Set size:", set.size);
  console.log("  Set has 2:", set.has(2));

  const map = new Map([["key1", "value1"], ["key2", "value2"]]);
  console.log("  Map size:", map.size);
  console.log("  Map get key1:", map.get("key1"));
  console.log("  Map keys:", Array.from(map.keys()));
} catch (e) {
  console.log("  Set/Map error:", e.message);
}

// Date
console.log("\n18. Date operations");
try {
  const now = new Date();
  console.log("  Current date:", now.toISOString());
  console.log("  Timestamp:", now.getTime());
  console.log("  Year:", now.getFullYear());
} catch (e) {
  console.log("  Date error:", e.message);
}

// Global functions
console.log("\n19. Global functions");
console.log("  parseInt:", parseInt("42"));
console.log("  parseFloat:", parseFloat("3.14"));
console.log("  isNaN:", isNaN("hello"));
console.log("  isFinite:", isFinite(100));

// Environment variables (requires --std flag)
console.log("\n20. Environment variables (std module)");
try {
  // Set environment variables
  std.setenv("TEST_VAR", "hello_world");
  std.setenv("TEST_NUMBER", "42");
  
  // Get environment variables
  console.log("  Get TEST_VAR:", std.getenv("TEST_VAR"));
  console.log("  Get TEST_NUMBER:", std.getenv("TEST_NUMBER"));
  console.log("  Get non-existent:", std.getenv("DOES_NOT_EXIST"));
  
  // Get all environment variables
  const environ = std.getenviron();
  console.log("  Environment has TEST_VAR:", "TEST_VAR" in environ);
  console.log("  Environment keys count:", Object.keys(environ).length > 0);
  
  // Unset environment variable
  std.unsetenv("TEST_VAR");
  console.log("  After unset TEST_VAR:", std.getenv("TEST_VAR"));
} catch (e) {
  console.log("  Environment error:", e.message);
}

// Extended JSON parsing (requires --std flag)
console.log("\n21. Extended JSON parsing (std module)");
try {
  // Check if parseExtJSON is available
  if (typeof std.parseExtJSON === 'function') {
    // Test standard JSON
    const standardJson = '{"name": "test", "value": 123}';
    const parsed1 = std.parseExtJSON(standardJson);
    console.log("  Standard JSON parsed:", parsed1.name === "test" && parsed1.value === 123);
    
    // Test JSON5 features - single quoted strings
    const singleQuotes = "{'name': 'test', 'value': 456}";
    const parsed2 = std.parseExtJSON(singleQuotes);
    console.log("  Single quotes:", parsed2.name === "test" && parsed2.value === 456);
    
    // Test JSON5 features - unquoted keys
    const unquotedKeys = '{name: "test", value: 789}';
    const parsed3 = std.parseExtJSON(unquotedKeys);
    console.log("  Unquoted keys:", parsed3.name === "test" && parsed3.value === 789);
    
    // Test JSON5 features - trailing commas
    const trailingComma = '{"name": "test", "value": 999,}';
    const parsed4 = std.parseExtJSON(trailingComma);
    console.log("  Trailing comma:", parsed4.name === "test" && parsed4.value === 999);
    
    // Test JSON5 features - comments
    const withComments = `{
      // This is a comment
      "name": "test",
      /* Multi-line
         comment */
      "value": 111
    }`;
    const parsed5 = std.parseExtJSON(withComments);
    console.log("  With comments:", parsed5.name === "test" && parsed5.value === 111);
    
    // Test JSON5 features - hexadecimal numbers
    const hexNumber = '{value: 0xFF}';
    const parsed6 = std.parseExtJSON(hexNumber);
    console.log("  Hex number (0xFF):", parsed6.value === 255);
    
    // Test JSON5 features - NaN and Infinity
    const specialNumbers = '{nan: NaN, inf: Infinity, negInf: -Infinity}';
    const parsed7 = std.parseExtJSON(specialNumbers);
    console.log("  NaN:", isNaN(parsed7.nan));
    console.log("  Infinity:", parsed7.inf === Infinity);
    console.log("  -Infinity:", parsed7.negInf === -Infinity);
  } else {
    console.log("  parseExtJSON not available in this WASI build");
    
    // Test standard JSON.parse as fallback
    const standardJson = '{"name": "test", "value": 123}';
    const parsed = JSON.parse(standardJson);
    console.log("  Standard JSON.parse works:", parsed.name === "test" && parsed.value === 123);
  }
  
} catch (e) {
  console.log("  Extended JSON error:", e.message);
}

console.log("\n=== Test Complete ===")
