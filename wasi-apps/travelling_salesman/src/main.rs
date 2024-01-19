/// Simple Rust binary implementing the [Travelling Salesman Problem (TSP)](https://en.wikipedia.org/wiki/Travelling_salesman_problem)
/// using the deterministic brute-force algorithm from the [`travelling_salesman` crate](https://docs.rs/travelling_salesman/latest/travelling_salesman/).
// TODO: select dataset with argv argument

mod datasets;
use datasets::{ City, CityConst, WG59, xy, CityRecord };

fn main() {

    // collect commandline arguments
    let args: Vec<String> = std::env::args().collect();

    // tsp gen [n] – print a random CSV for later
    if args.len() == 3 && args[1] == "write" {
        let n = args[2].parse::<usize>().unwrap();
        return write(&random_slice(&WG59, n));
    }

    // tsp rand [n] – solve a random selection
    if args.len() == 3 && args[1] == "rand" {
        let n = args[2].parse::<usize>().unwrap();
        return solve(&random_slice(&WG59, n));
    }

    // tsp read – read a previously generated CSV and solve it
    if args.len() == 2 && args[1] == "read" {
        return solve(&read());
    }

    // unknown or missing arguments
    eprintln!("unknown arguments! tsp {{ write [n] | rand [n] | read }}");
    std::process::exit(1);

}

/// Read in a CSV file with `x,y,name` and run the `travelling_salesman` solver.
fn read() -> Vec<City>{
    // read from stdin
    let mut reader = csv::ReaderBuilder::new().has_headers(false).from_reader(std::io::stdin());
    // collect cities from csv reader
    let mut cities: Vec<City> = vec![];
    for result in reader.deserialize() {
        let r: CityRecord = result.unwrap();
        cities.push(((r.x, r.y), r.name));
    }
    cities
}

/// Generate a CSV file with `x,y,name` for later consumption from the given slice.
fn write (cities: &[City]) {
    // open writer on stdout
    let mut writer = csv::WriterBuilder::new().has_headers(false).from_writer(std::io::stdout());
    // iterate over cities in slice
    for city in cities {
        writer.serialize(CityRecord{ x: city.0.0, y: city.0.1, name: city.1.clone() }).unwrap();
    }
    writer.flush().unwrap();
}

/// Run the `travelling_salesman::brute_force` algorithm on the chosen slice of cities.
fn solve (cities: &[City]) {
    // find the optimal path
    let path = travelling_salesman::brute_force::solve(&xy(cities));
    // map the path to city names
    let names: Vec<String> = path.route.iter().map(|c| cities[*c].1.clone()).collect();
    // print result
    println!("Path distance: {}, route: {:?}", path.distance, names);
}

/// Pick a random selection of coordinates from a `(x,y): [(f64, f64)]` dataset.
fn random_slice (slice: &[CityConst], amount: usize) -> Vec<City> {
    use rand::seq::SliceRandom;
    use rand::thread_rng;
    let mut copy = slice.to_vec();
    let slice = copy.partial_shuffle(&mut thread_rng(), amount).0;
    slice.to_vec().iter().map(|r| (r.0, r.1.to_string())).collect()
}
