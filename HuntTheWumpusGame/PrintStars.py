def print_stars(wumpuses_number, time):
    rating = time * (1 - wumpuses_number / 20) / (60 * 15)
    if rating > 0.75:
        print("★")
    elif rating > 0.5:
        print("★★")
    elif rating > 0.3:
        print("★★★")
    elif rating > 0.15:
        print("★★★★")
    else:
        print("★★★★★")