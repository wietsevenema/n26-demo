#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.12"
# dependencies = []
# ///

import json
import random
import string
import re

def generate_unique_names(target_count=1000):
    # Base Blocklist
    blocklist = {
        "SHIT", "FUCK", "COCK", "DICK", "CUNT", "TWAT", "SLUT", "WHOR", "PISS", "ARSE",
        "DYKE", "FAGG", "KIKB", "NIGG", "COON", "GOOK", "KYKE", "SPIC", "WOPP", "HELL",
        "DAMN", "DILF", "MILF", "CRAP", "BITE", "BONE", "BONG", "BUTT", "CLIT", "COKE",
        "DAGO", "DONG", "DUCH", "FAGS", "FART", "FECK", "FELC", "FELL", "HOAR", "HORE", 
        "JERK", "JISM", "JIZZ", "KNOB", "KOCK", "KUTT", "LUST", "MUFF", "POOP", "PORN", 
        "PRIC", "PUBE", "PUSS", "SCAT", "SHAG", "SLAG", "SMUT", "TEAT", "TURD", "WANK", 
        "LULW", "GVDD", "MOFF", "NSAP", "NSAD", "TWAF", "TWZD", "SIFF", "SITH", "FCKR",
        "ARAA", "ORAN", "TRAN", "FRIG", "AUTH", "BAZE", "DISK", "SSLI", "TLSS", "DNSS",
        "SERG", "VITE", "WHIT", "YELL", "TEST", "BLAC", "BLUE", "GREE", "PURP", "PINK", 
        "REDD", "CYAN", "SILV", "GOLD", "JSIS", "SAML", "OIDC", "IAMM", "FSSS", "USER",
        "VILL", "CINT"
    }

    leet_map = str.maketrans("0134578", "OIELSTB")
    vowels = set("AEIOUY")
    non_vowel_digits = "256789"
    consonants = "".join([c for c in string.ascii_uppercase if c not in vowels])

    def are_similar(s1, s2):
        if len(s1) != len(s2):
            return False
        
        # 1. Hamming distance check (1 character difference)
        diffs = sum(1 for c1, c2 in zip(s1, s2) if c1 != c2)
        if diffs <= 1:
            return True
            
        # 2. Transposition check (e.g. SIHT vs SHIT)
        if sorted(s1) == sorted(s2):
            # If sorted strings are identical and distance is small (for 4 chars), 
            # it's likely a single swap.
            return True
            
        return False

    def is_safe(name):
        # Strip dashes and normalize
        candidate = name.upper().replace("-", "")
        # Get leet version for full check
        leet_candidate = candidate.translate(leet_map)
        
        for bad in blocklist:
            # Direct match
            if bad == candidate or bad == leet_candidate:
                return False
            # Fuzzy match (only for 4-char stems)
            if len(candidate) >= 4:
                stem = candidate[:4]
                if are_similar(stem, bad) or are_similar(leet_candidate[:4], bad):
                    return False
        return True

    # User-vetted nouns only
    vetted_nouns = [
        "ADMN", "AMPP", "APIS", "APPI", "ARBT", "AREA", "ARGO", "ASIC", 
        "ASTO", "ATOM", "BASE", "BASY", "BAUD", "BEAM", "BITT", "BOAT", 
        "BOHR", "BOND", "BOOT", "BOSH", "BQRY", "BRAS", "BRON", "BUNN", 
        "BYTE", "CAMP", "CARB", "CERE", "CERT", "CFUN", "CHIR", "CHPT", 
        "CIDR", "CLON", "CMAP", "CODE", "COLO", "CONN", "COPP", 
        "CORE", "COSM", "CRUI", "CRUN", "CRYP", "CSSS", "DARK", "DART", 
        "DASH", "DBAS", "DEBG", "DECC", "DENO", "DEST", "DEVI", "DEVS", 
        "DIFF", "DIVV", "DOME", "DOOR", "DRIV", "DROI", "EDGE", "EINS", 
        "ELEV", "EMAC", "ERIS", "ERRO", "FILE", "FISH", "FLEB", "FLEX", 
        "FLOW", "FLOX", "FLUX", "FOLD", "FORC", "FORK", "FORT", "FPGA", 
        "GAEE", "GATE", "GCEE", "GCSB", "GEMI", "GITT", "GKEE", "GLOW", 
        "GOLD", "GOOO", "GOPH", "GPUU", "GRAV", "GRID", "GRPC", "GUID", 
        "GUIL", "HALO", "HASH", "HAUM", "HAWK", "HCLL", "HDDD", "HEAD", 
        "HELM", "HERT", "HOLD", "HOME", "HOST", "HPAS", "HTML", "HTMX", 
        "HTTP", "HUBB", "HYPR", "IDEA", "INFO", "INGR", "IONS", 
        "IRON", "JACK", "JOBB", "JSON", "JUMP", "JUPI", "KEPL", 
        "KERN", "KEYY", "KVSS", "LANN", "LASR", "LEAD", "LENS", "LIFT", 
        "LINK", "LLMM", "LOAD", "LOGG", "LOGS", "LYRA", "MACH", "MAGN", 
        "MAKE", "MARS", "MASK", "MASR", "MASS", "MAST", "MECH", "MEDD", 
        "MELL", "MERC", "MERG", "MESH", "METR", "MIRR", "MOLL", 
        "MOON", "NAME", "NEED", "NEPT", "NEWT", "NEXT", "NICC", "NODE", 
        "NOVA", "NPUU", "NUXT", "OHMM", "OIDC", "ONYX", "PAGE", "PALM", 
        "PASS", "PATH", "PEER", "PILO", "PIMY", "PINK", "PIPE", "PLAT", 
        "PLUG", "PLUT", "PODS", "POOL", "PORT", "PROB", "PROD", "PROT", 
        "PRSM", "PUBS", "PULL", "PULM", "PULS", "PUPP", "PURP", "PUSH", 
        "PVCC", "PVOL", "QUAN", "QUAS", "RAYY", "REAC", "ROMS", "ROOF", 
        "ROOM", "ROOT", "ROSE", "ROUT", "RUST", "SAGA", "SALT", "SATU", 
        "SCAN", "SECC", "SENS", "SERG", "SHEW", "SHIP", "SIGN", "SILV", 
        "SITE", "SLIP", "SNAP", "SPAC", "SPAN", "SRES", "SSDD", "STAG", 
        "STAI", "STAR", "STAT", "STEE", "STEP", "STEY", "STRE", "SUDO", 
        "SVCS", "SYNC", "TAGG", "TAIL", "TALL", "TASK", "TERA", "TESS", 
        "TEST", "TIER", "TIME", "TIMO", "TINN", "TOML", "TOOL", "TPUU", 
        "URAN", "URLL", "USER", "UUID", "VANE", "VANT", "VDII", "VELL", "VENT", "VENU", 
        "VILL", "VIMM", "VITE", "VOLL", "VOLT", "VPAS", "VPCC", "VPNN", "VPUU", "VSCD", 
        "WALL", "WANN", "WARN", "WARP", "WASM", "WATT", "WAVE", "WAYY", "WEBB", 
        "WIFI", "WIRE", "WORD", "WORK", "YAML", "YELL", "ZEDD", "ZINC", "ZONE", "ZSHH"

    ]

    unique_names = set()
    random.seed(42)
    
    # 1. Add vetted nouns + digit
    for word in vetted_nouns:
        for digit in non_vowel_digits:
            candidate = f"{word}-{digit}"
            if is_safe(candidate):
                unique_names.add(candidate)
    
    # 2. Add randoms if needed
    while len(unique_names) < target_count:
        stem = "".join(random.choices(consonants, k=4))
        digit = random.choice(non_vowel_digits)
        candidate = f"{stem}-{digit}"
        if is_safe(candidate):
            unique_names.add(candidate)
    
    final_list = list(unique_names)
    random.shuffle(final_list)
    final_list = final_list[:target_count]
    
    print(f"Aggressive Safety Validation Passed: {len(final_list)} unique IDs.")
    return [{"name": name} for name in final_list]

if __name__ == "__main__":
    results = generate_unique_names(1000)
    with open('backend/instance-ids.json', 'w') as f:
        json.dump(results, f, indent=2)
    print("backend/instance-ids.json written successfully.")
