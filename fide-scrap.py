#!/usr/bin/python

import string
import requests
import bs4


def parse_player_calculations(player_name, calculationsHtml):
	player_elo = 0
	event = "-"
	site = "-"
	date = "-"
	round = 1
	white = "-"
	black = "-"
	result = "*"
	white_elo = "-"
	black_elo = "-"
	games = []
	
	soup = bs4.BeautifulSoup(calculationsHtml)
	main_col = soup.find(id = 'main-col')
	contents_table = main_col.find_all('table', recursive=False)[1]
	contents_row = contents_table.tbody.find_all('tr', recursive=False)[1]
	if contents_row.table == None:
		return []
	actions_table = contents_row.find_all('table', recursive=True)[1]
	actions = actions_table.tbody.find_all('tr', recursive=False)
	for action in actions:
		attrs = action.contents
		if len(attrs) == 11: #game
			opponent_name = attrs[0].get_text().strip()
			opponent_elo = attrs[3].get_text().strip()
			i = string.find(opponent_elo, '*')
			if i >= 0:
				opponent_elo = opponent_elo[:i].strip()
			if opponent_elo == '':
				opponent_elo = '-'
			player_points = attrs[5].get_text().strip()
			player_is_white = string.find(attrs[0].img['src'], 'clr_wh') >= 0
			if player_is_white:
				white, black = player_name, opponent_name
				white_elo, black_elo = player_elo, opponent_elo
				player_points = attrs[5].get_text().strip()
				if player_points == '0.50':
					result = '1/2-1/2'
				elif player_points == '1.00':
					result = '1-0'
				elif player_points == '0.00':
					result = '0-1'
				else:
					result = '*'
			else:
				white, black = opponent_name, player_name
				white_elo, black_elo = opponent_elo, player_elo
				if player_points == '0.50':
					result = '1/2-1/2'
				elif player_points == '1.00':
					result = '0-1'
				elif player_points == '0.00':
					result = '1-0'
				else:
					result = '*'
			t = (event, site, date, round, white, white_elo, black, black_elo, result)
			games.append(t)
			round = round + 1
			pass
		elif len(attrs) == 4: #tournament
			event = attrs[0].get_text().strip()
			site = attrs[1].get_text().strip()
			date = attrs[3].get_text().strip()
			round = 1
			pass
		elif len(attrs) == 8: #summary
			ro = attrs[1].get_text().strip()
			if ro == 'Ro':
				pass
			else:
				player_elo = ro
		else:
			pass
	return games



def scrap_player_calculations(player_name, period_info):
	resp = requests.get('http://ratings.fide.com/individual_calculations.phtml', params = period_info)
	if resp.status_code == requests.codes.ok:
		html = resp.text
		games = parse_player_calculations(player_name, html)
		return games
	else:
		resp.raise_for_status()



def main():
	games = []
	for year in xrange(2007, 2014):
		for month in xrange(1, 13):
			period_info = {
				'idnumber': 4206045,
				'rating_period': '%04d-%02d-01' % (year, month)
			}
			t = scrap_player_calculations('Anastasopoulos, Spyros', period_info)
			games.extend(t)
			for game in t:
				print "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s" % game





if __name__ == '__main__':
	main()
