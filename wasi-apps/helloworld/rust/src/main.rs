fn main() {
  println!("This is Rust.");

  // print commandline arguments
  let args: Vec<String> = std::env::args().collect();
  let name = args[0].clone();
  println!("file '{name}' was called with arguments: {args:?}");

  // print environment variables
  if args.contains(&String::from("print_envs")) {
    println!("environment variables:");
    for (key, value) in std::env::vars() {
      println!(" - {}: {}", key, value);
    };
  }

  // list root filesystem
  if args.contains(&String::from("print_rootfs")) {
    match std::fs::read_dir("/") {
      Err(e) => println!("ERR: couldn't open directory '/': {e}"),
      Ok(iter) => {
        println!("listing '/' contents:");
        for item in iter {
          println!(" - {}", item.unwrap().path().display());
        }
      }
    };
  }

  // get filename to try a few operations on
  if let Some(file) = args.iter().find(|&arg| arg.starts_with("file:")) {
    let filename = file.strip_prefix("file:").unwrap();

    // truncate when the file is getting large
    match std::fs::metadata(filename) {
      Err(e) => println!("ERR: can't get metadata for file: {e}"),
      Ok(m) => {
        println!("{filename} has {} bytes", m.len());
        if m.len() > 256 {
          match std::fs::File::options().write(true).truncate(true).open(filename) {
            Err(e) => println!("ERR: couldn't truncate file: {e}"),
            Ok(_) => println!("256 bytes ought to be enough for anyone!"),
          }
        }
      },
    }
  
    // append line to the file
    use std::io::prelude::*;
    match std::fs::File::options().append(true).open(filename) {
      Err(e) => println!("ERR: couldn't open the file: {e}"),
      Ok(mut file) => {
        let epoch = std::time::SystemTime::now().duration_since(std::time::UNIX_EPOCH).unwrap().as_secs();
        if let Err(e) = writeln!(file, "Seconds since UNIX_EPOCH: {epoch}") {
          println!("ERR: failed to write to file: {e}");
        }
      },
    }
  
    // read contents of the file
    match std::fs::read(filename) {
      Err(e) => println!("ERR: couldn't open file for reading: {e}"),
      Ok(content) => {
        let text = String::from_utf8_lossy(&content);
        println!("'{filename}':\n{text}");
      },
    }
  }

}
