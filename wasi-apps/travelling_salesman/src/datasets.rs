
/// The `City` type combines an `(x, y)` float coordinate tuple with a name.
pub type City = ((f64, f64), String);

/// The `CityConst` is similar to `City`, but with static strings.
pub type CityConst = ((f64, f64), &'static str);

/// The `CityRecord` is a serializable type for CSV storage.
#[derive(Debug, serde::Serialize, serde::Deserialize)]
pub struct CityRecord { pub x: f64, pub y: f64, pub name: String }

/// Helper function to map a vector of `City`s to their coordinates.
pub fn xy(cities: &[City]) -> Vec<(f64, f64)> {
  cities.iter().map(|c| c.0).collect()
}

// The following datasets are taken from: https://people.sc.fsu.edu/~jburkardt/datasets/cities/cities.html

#[allow(dead_code)]
/// Dataset with `(x,y)` coordinates of 59 West German cities.
/// > WG59 describes 59 cities in West Germany, and comes from Spaeth.
pub static WG59: [CityConst; 59] = [
    ((  54.0, -65.0 ), "Augsburg" ),
    ((   0.0,  71.0 ), "Bielefeld" ),
    (( -31.0,  53.0 ), "Bochum" ),
    ((   8.0, 111.0 ), "Bremen" ),
    ((   1.0,  -9.0 ), "Darmstadt" ),
    (( -36.0,  52.0 ), "Essen" ),
    (( -22.0, -76.0 ), "Freiburg" ),
    ((   0.0,  20.0 ), "Giessen" ),
    ((  34.0, 129.0 ), "Hamburg" ),
    ((  28.0,  84.0 ), "Hannover" ),
    ((  12.0, -38.0 ), "Heilbronn" ),
    (( -21.0, -26.0 ), "Kaiserslautern" ),
    ((  -6.0, -41.0 ), "Karlsruhe" ),
    ((  21.0,  45.0 ), "Kassel" ),
    ((  38.0, -90.0 ), "Kempten" ),
    (( -24.0,  10.0 ), "Koblenz" ),
    (( -38.0,  35.0 ), "Koeln" ),
    ((  86.0, -57.0 ), "Landshut" ),
    ((  58.0,  -1.0 ), "Lichtenfels" ),
    ((  -9.0,  -3.0 ), "Mainz" ),
    ((  70.0, -74.0 ), "Muenchen" ),
    (( -20.0,  70.0 ), "Muenster" ),
    (( -43.0,  44.0 ), "Neuss" ),
    ((  59.0, -26.0 ), "Nuernburg" ),
    ((  -5.0, 114.0 ), "Oldenburg" ),
    ((  83.0, -41.0 ), "Regensburg" ),
    ((  27.0, 153.0 ), "Rendsburg" ),
    ((  12.0, -49.0 ), "Stuttgart" ),
    ((  30.0, -65.0 ), "Ulm" ),
    ((  31.0, -12.0 ), "Wuerzburg" ),
    (( -57.0,  28.0 ), "Aachen" ),
    ((  44.0, -28.0 ), "Ansbach" ),
    ((   7.0,  -7.0 ), "Aschaffenburg" ),
    ((  54.0,  -8.0 ), "Bamberg" ),
    ((  65.0,  -8.0 ), "Bayreuth" ),
    (( -35.0,  25.0 ), "Bonn" ),
    ((  46.0,  79.0 ), "Braunschweig" ),
    ((   5.0, 118.0 ), "Bremen" ),
    ((  56.0,   4.0 ), "Coburg" ),
    (( -21.0,  54.0 ), "Dortmund" ),
    (( -40.0,  45.0 ), "Duesseldorf" ),
    (( -43.0,  51.0 ), "Duisburg" ),
    ((  57.0, -21.0 ), "Erlangen" ),
    ((   0.0,   0.0 ), "Frankfurt" ),
    ((  25.0,  15.0 ), "Fulda" ),
    ((  56.0, -25.0 ), "Fuerth" ),
    (( -34.0,  56.0 ), "Gelsen-Kirchen" ),
    (( -24.0,  36.0 ), "Gummersburg" ),
    (( -25.0,  49.0 ), "Hagen" ),
    ((  64.0, -26.0 ), "Hersbruck" ),
    ((  63.0, -48.0 ), "Ingolstadt" ),
    ((  37.0, 155.0 ), "Kiel" ),
    ((  -5.0, -24.0 ), "Mannheim" ),
    ((   2.0,  28.0 ), "Marburg" ),
    (( -18.0, -58.0 ), "Offenburg" ),
    (( -10.0,  82.0 ), "Osnabrueck" ),
    ((  12.0, -58.0 ), "Reutlingen" ),
    (( -40.0, -28.0 ), "Saarbruecken" ),
    (( -16.0,  28.0 ), "Siegen" ),
];

#[allow(dead_code)]
/// Dataset with `(x,y)` coordinates of 128 North American cities.
/// > SGB128 describes 128 cities in North America.
pub const SGB128: [CityConst; 128] = [
    (( -5572.57,  2839.81 ), "Youngstown, OH" ),
    (( -6729.21,  2962.82 ), "Yankton, SD" ),
    (( -8326.72,  3219.84 ), "Yakima, WA" ),
    (( -4961.07,  2920.67 ), "Worcester, MA" ),
    (( -6202.70,  3014.64 ), "Wisconsin Dells, WI" ),
    (( -5544.93,  2494.33 ), "Winston-Salem, NC" ),
    (( -6712.64,  3446.49 ), "Winnipeg, MB" ),
    (( -5400.52,  2707.84 ), "Winchester, VA" ),
    (( -5383.92,  2365.84 ), "Wilmington, NC" ),
    (( -5220.18,  2746.55 ), "Wilmington, DE" ),
    (( -7159.69,  3326.96 ), "Williston, ND" ),
    (( -5320.37,  2850.20 ), "Williamsport, PA" ),
    (( -5685.17,  2603.52 ), "Williamson, WV" ),
    (( -6805.21,  2342.34 ), "Wichita Falls, TX" ),
    (( -6725.75,  2604.20 ), "Wichita, KS" ),
    (( -5577.40,  2768.64 ), "Wheeling, WV" ),
    (( -5531.11,  1846.22 ), "West Palm Beach, FL" ),
    (( -8313.57,  3276.50 ), "Wenatchee, WA" ),
    (( -8456.60,  2861.92 ), "Weed, CA" ),
    (( -5690.01,  2157.15 ), "Waycross, GA" ),
    (( -6193.72,  3106.52 ), "Wausau, WI" ),
    (( -6068.67,  2926.89 ), "Waukegan, IL" ),
    (( -6709.88,  3102.40 ), "Watertown, SD" ),
    (( -5245.72,  3038.81 ), "Watertown, NY" ),
    (( -6380.27,  2936.57 ), "Waterloo, IA" ),
    (( -5047.44,  2870.91 ), "Waterbury, CT" ),
    (( -5322.42,  2687.11 ), "Washington, DC" ),
    (( -5468.21,  2891.63 ), "Warren, PA" ),
    (( -8176.09,  3183.22 ), "Walla Walla, WA" ),
    (( -6711.93,  2179.95 ), "Waco, TX" ),
    (( -6047.92,  2672.62 ), "Vincennes, IN" ),
    (( -6702.97,  1990.63 ), "Victoria, TX" ),
    (( -6279.39,  2235.23 ), "Vicksburg, MS" ),
    (( -8507.06,  3404.34 ), "Vancouver, BC" ),
    (( -6772.07,  3241.95 ), "Valley City, ND" ),
    (( -5754.27,  2130.20 ), "Valdosta, GA" ),
    (( -5198.07,  2978.71 ), "Utica, NY" ),
    (( -5509.00,  2756.92 ), "Uniontown, PA" ),
    (( -6584.82,  2235.23 ), "Tyler, TX" ),
    (( -7909.38,  2940.71 ), "Twin Falls, ID" ),
    (( -6050.69,  2294.65 ), "Tuscaloosa, AL" ),
    (( -6129.46,  2367.20 ), "Tupelo, MS" ),
    (( -6626.97,  2498.48 ), "Tulsa, OK" ),
    (( -7667.55,  2226.26 ), "Tucson, AZ" ),
    (( -7221.19,  2568.27 ), "Trinidad, CO" ),
    (( -5166.26,  2779.70 ), "Trenton, NJ" ),
    (( -5916.64,  3092.70 ), "Traverse City, MI" ),
    (( -5484.79,  3016.03 ), "Toronto, ON" ),
    (( -6610.36,  2698.17 ), "Topeka, KS" ),
    (( -5772.25,  2877.83 ), "Toledo, OH" ),
    (( -6498.45,  2309.87 ), "Texarkana, TX" ),
    (( -6039.65,  2727.21 ), "Terre Haute, IN" ),
    (( -5696.92,  1931.22 ), "Tampa, FL" ),
    (( -5823.36,  2103.96 ), "Tallahassee, FL" ),
    (( -8459.38,  3264.08 ), "Tacoma, WA" ),
    (( -5261.63,  2974.55 ), "Syracuse, NY" ),
    (( -5689.32,  2252.50 ), "Swainsboro, GA" ),
    (( -5551.82,  2343.71 ), "Sumter, SC" ),
    (( -5195.30,  2832.23 ), "Stroudsburg, PA" ),
    (( -8380.61,  2622.85 ), "Stockton, CA" ),
    (( -6188.88,  3076.14 ), "Stevens Point, WI" ),
    (( -5570.49,  2788.70 ), "Steubenville, OH" ),
    (( -7132.05,  2806.65 ), "Sterling, CO" ),
    (( -5463.38,  2636.00 ), "Staunton, VA" ),
    (( -5790.89,  2758.28 ), "Springfield, OH" ),
    (( -6445.93,  2571.74 ), "Springfield, MO" ),
    (( -5015.64,  2908.91 ), "Springfield, MA" ),
    (( -6194.43,  2749.99 ), "Springfield, IL" ),
    (( -8112.52,  3293.77 ), "Spokane, WA" ),
    (( -5959.50,  2879.91 ), "South Bend, IN" ),
    (( -6683.62,  3008.43 ), "Sioux Falls, SD" ),
    (( -6660.11,  2935.87 ), "Sioux City, IA" ),
    (( -6477.72,  2246.28 ), "Shreveport, LA" ),
    (( -6675.33,  2324.36 ), "Sherman, TX" ),
    (( -7390.45,  3095.47 ), "Sheridan, WY" ),
    (( -6680.17,  2434.22 ), "Seminole, OK" ),
    (( -6012.69,  2240.06 ), "Selma, AL" ),
    (( -6441.79,  2674.67 ), "Sedalia, MO" ),
    (( -8452.47,  3288.93 ), "Seattle, WA" ),
    (( -5228.45,  2861.23 ), "Scranton, PA" ),
    (( -7162.46,  2893.02 ), "Scottsbluff, NB" ),
    (( -5109.61,  2958.66 ), "Schenectady, NY" ),
    (( -5602.95,  2216.59 ), "Savannah, GA" ),
    (( -5828.20,  3212.26 ), "Sault Sainte Marie, MI" ),
    (( -5702.45,  1889.08 ), "Sarasota, FL" ),
    (( -8479.42,  2656.02 ), "Santa Rosa, CA" ),
    (( -7320.67,  2465.33 ), "Santa Fe, NM" ),
    (( -8270.73,  2378.25 ), "Santa Barbara, CA" ),
    (( -8144.31,  2332.65 ), "Santa Ana, CA" ),
    (( -8421.36,  2580.03 ), "San Jose, CA" ),
    (( -8458.67,  2610.42 ), "San Francisco, CA" ),
    (( -5714.88,  2864.02 ), "Sandusky, OH" ),
    (( -8094.56,  2260.10 ), "San Diego, CA" ),
    (( -8105.59,  2356.85 ), "San Bernardino, CA" ),
    (( -6805.92,  2032.79 ), "San Antonio, TX" ),
    (( -6939.97,  2173.73 ), "San Angelo, TX" ),
    (( -7730.40,  2816.32 ), "Salt Lake City, UT" ),
    (( -5223.61,  2651.18 ), "Salisbury, MD" ),
    (( -8405.49,  2533.72 ), "Salinas, CA" ),
    (( -6744.43,  2683.68 ), "Salina, KS" ),
    (( -7324.14,  2662.24 ), "Salida, CO" ),
    (( -8500.82,  3105.14 ), "Salem, OR" ),
    (( -6432.79,  3105.85 ), "Saint Paul, MN" ),
    (( -6231.74,  2668.46 ), "Saint Louis, MO" ),
    (( -6553.01,  2747.93 ), "Saint Joseph, MO" ),
    (( -5975.39,  2908.91 ), "Saint Joseph, MI" ),
    (( -4976.25,  3069.21 ), "Saint Johnsbury, VT" ),
    (( -6506.72,  3148.67 ), "Saint Cloud, MN" ),
    (( -5618.84,  2065.25 ), "Saint Augustine, FL" ),
    (( -5799.89,  3000.82 ), "Saginaw, MI" ),
    (( -8394.41,  2666.40 ), "Sacramento, CA" ),
    (( -5041.91,  3013.26 ), "Rutland, VT" ),
    (( -7222.55,  2307.80 ), "Roswell, NM" ),
    (( -5375.64,  2483.28 ), "Rocky Mount, NC" ),
    (( -7547.32,  2873.69 ), "Rock Springs, WY" ),
    (( -6156.40,  2920.67 ), "Rockford, IL" ),
    (( -5362.51,  2982.15 ), "Rochester, NY" ),
    (( -6388.57,  3041.59 ), "Rochester, MN" ),
    (( -5523.51,  2575.20 ), "Roanoke, VA" ),
    (( -5351.44,  2593.85 ), "Richmond, VA" ),
    (( -5865.51,  2752.08 ), "Richmond, IN" ),
    (( -7744.91,  2678.84 ), "Richfield, UT" ),
    (( -6178.52,  3153.51 ), "Rhinelander, WI" ),
    (( -8278.33,  2730.66 ), "Reno, NV" ),
    (( -7230.86,  3483.78 ), "Regina, SA" ),
    (( -8446.23,  2776.26 ), "Red Bluff, CA" ),
    (( -5246.43,  2786.63 ), "Reading, PA" ),
    (( -5613.31,  2843.96 ), "Ravenna, OH" ),
];