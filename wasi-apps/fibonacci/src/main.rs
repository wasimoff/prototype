use std::process::exit;

// this is stupid fast on my machine and easily overflows the
// u128 (after fib(186)) before taking a perceivable amount of time
pub fn fibonacci (n: i32) -> u128 {
  if n <= 0 { return 0 }
  if n == 1 { return 1 }
  let mut fib = 1;
  let mut prev = 0;
  for _ in 1..n {
    (prev, fib) = (fib, fib + prev)
  }
  fib
}

// this is horribly inefficient but creates some load
pub fn fibonacci_recursive (n: i32) -> i128 {
  if n <= 0 { return 0 }
  if n == 1 { return 1 }
  fibonacci_recursive(n-1) + fibonacci_recursive(n-2)
}

fn main() {

  // get rank from commandline arguments
  let args: Vec<String> = std::env::args().collect();
  if args.len() < 2 {
    eprintln!("fibonacci rank u32 required in first argument!");
    exit(1);
  }
  let rank = args[1].parse::<i32>().unwrap();

  let fib = fibonacci_recursive(rank);
  println!("{}", fib);

}
