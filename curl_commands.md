# Eval Image API Curl Commands

Run these from the repository root after starting the stack with `make dev`.

## Generated Rye Whiskey

```sh
curl -sS -X PUT http://localhost:8080/api/applications/generated-rye-whiskey \
  -H 'Content-Type: application/json' \
  -d '{
    "Brand": "POM CREEK DISTILLING COMPANY, LLC",
    "ClassType": "Rye Whiskey",
    "NetContents": "750 mL",
    "ABV": "45% ALC/VOL (90 Proof)",
    "GovernmentWarning": "GOVERNMENT WARNING",
    "NameAndAddress": "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA",
    "ForeignBlocks": [],
    "DeclaredLanguages": ["en"]
  }'

curl -sS -X POST http://localhost:8080/api/verify \
  -F application_id=generated-rye-whiskey \
  -F image=@evals/images/generated_rye_whiskey.png
```

## Generated Blueberry Liqueur

```sh
curl -sS -X PUT http://localhost:8080/api/applications/generated-blueberry-liqueur \
  -H 'Content-Type: application/json' \
  -d '{
    "Brand": "POM CREEK DISTILLING COMPANY, LLC",
    "ClassType": "Blueberry Liqueur/Cordial",
    "NetContents": "375 mL",
    "ABV": "25% ALC/VOL (50 Proof)",
    "GovernmentWarning": "GOVERNMENT WARNING",
    "NameAndAddress": "POM CREEK DISTILLING COMPANY, LLC LOUDOUN COUNTY, VA",
    "ForeignBlocks": [],
    "DeclaredLanguages": ["en"]
  }'

curl -sS -X POST http://localhost:8080/api/verify \
  -F application_id=generated-blueberry-liqueur \
  -F image=@evals/images/generated_blueberry_liqueur.png
```

## Generated Apple Brandy

```sh
curl -sS -X PUT http://localhost:8080/api/applications/generated-apple-brandy \
  -H 'Content-Type: application/json' \
  -d '{
    "Brand": "POM CREEK DISTILLING COMPANY, LLC",
    "ClassType": "Apple Brandy",
    "NetContents": "750 mL",
    "ABV": "40% ALC/VOL (80 Proof)",
    "GovernmentWarning": "GOVERNMENT WARNING",
    "NameAndAddress": "POM CREEK DISTILLING COMPANY, LLC LOUDOUN COUNTY, VA",
    "ForeignBlocks": [],
    "DeclaredLanguages": ["en"]
  }'

curl -sS -X POST http://localhost:8080/api/verify \
  -F application_id=generated-apple-brandy \
  -F image=@evals/images/generated_apple_brandy.png
```

## Generated Bourbon With Contents

```sh
curl -sS -X PUT http://localhost:8080/api/applications/generated-bourbon-with-contents \
  -H 'Content-Type: application/json' \
  -d '{
    "Brand": "POM BOURBON",
    "ClassType": "Bourbon",
    "NetContents": "750 mL",
    "ABV": "45% ALC/VOL (90 Proof)",
    "GovernmentWarning": "GOVERNMENT WARNING",
    "NameAndAddress": "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA",
    "ForeignBlocks": [],
    "DeclaredLanguages": ["en"]
  }'

curl -sS -X POST http://localhost:8080/api/verify \
  -F application_id=generated-bourbon-with-contents \
  -F image=@evals/images/generated_bourbon_with_contents.png
```

## Generated Bourbon No Contents

```sh
curl -sS -X PUT http://localhost:8080/api/applications/generated-bourbon-no-contents \
  -H 'Content-Type: application/json' \
  -d '{
    "Brand": "POM BOURBON",
    "ClassType": "Bourbon",
    "NetContents": "750 mL",
    "ABV": "45% ALC/VOL (90 Proof)",
    "GovernmentWarning": "GOVERNMENT WARNING",
    "NameAndAddress": "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA",
    "ForeignBlocks": [],
    "DeclaredLanguages": ["en"]
  }'

curl -sS -X POST http://localhost:8080/api/verify \
  -F application_id=generated-bourbon-no-contents \
  -F image=@evals/images/generated_bourbon_no_contents.png
```

## Generated Bourbon Government Warning No Contents

```sh
curl -sS -X PUT http://localhost:8080/api/applications/generated-bourbon-gov-warning-no-contents \
  -H 'Content-Type: application/json' \
  -d '{
    "Brand": "POM BOURBON",
    "ClassType": "Bourbon",
    "NetContents": "750 mL",
    "ABV": "45% ALC/VOL (90 Proof)",
    "GovernmentWarning": "GOVERNMENT WARNING",
    "NameAndAddress": "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA",
    "ForeignBlocks": [],
    "DeclaredLanguages": ["en"]
  }'

curl -sS -X POST http://localhost:8080/api/verify \
  -F application_id=generated-bourbon-gov-warning-no-contents \
  -F image=@evals/images/generated_bourbon_gov_warning_no_contents.png
```

## Generated Bourbon Government Warning With Contents

```sh
curl -sS -X PUT http://localhost:8080/api/applications/generated-bourbon-gov-warning-with-contents \
  -H 'Content-Type: application/json' \
  -d '{
    "Brand": "POM BOURBON",
    "ClassType": "Bourbon",
    "NetContents": "750 mL",
    "ABV": "45% ALC/VOL (90 Proof)",
    "GovernmentWarning": "GOVERNMENT WARNING",
    "NameAndAddress": "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA",
    "ForeignBlocks": [],
    "DeclaredLanguages": ["en"]
  }'

curl -sS -X POST http://localhost:8080/api/verify \
  -F application_id=generated-bourbon-gov-warning-with-contents \
  -F image=@evals/images/generated_bourbon_gov_warning_with_contents.png
```

## Generated Bourbon No Government Warning

```sh
curl -sS -X PUT http://localhost:8080/api/applications/generated-bourbon-no-gov-warning \
  -H 'Content-Type: application/json' \
  -d '{
    "Brand": "POM BOURBON",
    "ClassType": "Bourbon",
    "NetContents": "750 mL",
    "ABV": "45% ALC/VOL (90 Proof)",
    "GovernmentWarning": "GOVERNMENT WARNING",
    "NameAndAddress": "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA",
    "ForeignBlocks": [],
    "DeclaredLanguages": ["en"]
  }'

curl -sS -X POST http://localhost:8080/api/verify \
  -F application_id=generated-bourbon-no-gov-warning \
  -F image=@evals/images/generated_bourbon_no_gov_warning.jpg
```

## Generated Tequila

```sh
curl -sS -X PUT http://localhost:8080/api/applications/generated-tequila \
  -H 'Content-Type: application/json' \
  -d '{
    "Brand": "POM BLANCO TEQUILA",
    "ClassType": "Blanco Tequila",
    "NetContents": "750 mL",
    "ABV": "40% ALC/VOL (80 Proof)",
    "GovernmentWarning": "GOVERNMENT WARNING",
    "NameAndAddress": "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA",
    "ForeignBlocks": [],
    "DeclaredLanguages": ["en"]
  }'

curl -sS -X POST http://localhost:8080/api/verify \
  -F application_id=generate-tequila \
  -F image=@evals/images/generated-tequila.png
```
