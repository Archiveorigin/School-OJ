# Ranking and submission UI behaviors

- Scoring rule is creation-time data. Missing or invalid input resolves to `penalty`; valid values are `penalty` and `score`.
- Published mode preserves server order/ranks. Performance mode locally orders penalty rows by solved descending then penalty ascending, and score rows by total score descending then full-score count descending.
- The penalty and score tables are separate render branches selected from `LeaderboardData.scoringRule`.
- Penalty mode uses the first accepted submission and adds 20 minutes for each completed failure before it; queueing, judging, pending review, and system failures do not add attempts or penalty. In manual-review exams, the first submission awarded the problem's full score is the accepted submission.
- Header cells remain sticky at the top of the scoreboard scroll viewport.
- Result cells retain the extracted 150 ms hover transition and use accepted green, wrong/partial red, pending blue, and no-submission gray.
- The shared controls wrap below approximately 1100 px. The scoreboard itself never collapses columns; it scrolls horizontally.
- Participant filtering matches names only and never recalculates award membership.
- Award cutoffs use the actual unfiltered participant count and cumulative ceiling thresholds. Default gold/silver/bronze values are 10% each; contest managers can change them while their sum remains at most 100%.
- Rows display only a participant name; student number, email, avatar, organization, team and submission metadata are absent.
- Submission records open the code dialog by row click or the explicit “查看代码” action.
- The copy action copies the exact source text and reports success/failure.
- The verdict graphic is semantic (`role="img"` plus an accessible label) and maps accepted to green, queued/running to blue, and all completed non-accepted states to red.
- The source panel scrolls rather than expanding beyond the dialog viewport.
- The problem page continues receiving live judge events for data freshness, but no longer renders the obsolete bottom result strip.
