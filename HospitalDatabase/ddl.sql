CREATE SCHEMA hospital;
SET search_path = hospital;

CREATE TABLE Doctors (
    doctor_id SERIAL PRIMARY KEY,
    surname VARCHAR(100) NOT NULL,
    name VARCHAR(100) NOT NULL,
    birth_date DATE NOT NULL,
    sex VARCHAR(100) NOT NULL,
    specialization VARCHAR(100) NOT NULL
);

CREATE TABLE Patients (
    patient_id SERIAL PRIMARY KEY,
    surname VARCHAR(100) NOT NULL,
    name VARCHAR(100) NOT NULL,
    birth_date DATE NOT NULL,
    sex VARCHAR(100) NOT NULL,
    phone_number VARCHAR(15)
);

CREATE TABLE Appointments (
    appointment_id SERIAL PRIMARY KEY,
    doctor_id INTEGER REFERENCES Doctors(doctor_id),
    patient_id INTEGER REFERENCES Patients(patient_id),
    date_time TIMESTAMP NOT NULL,
    duration INTEGER NOT NULL,
    office INTEGER NOT NULL
);

CREATE TABLE Conclusions (
    conclusion_id SERIAL PRIMARY KEY,
    appointment_id INTEGER REFERENCES Appointments(appointment_id),
    complaints TEXT NOT NULL,
    examination_results TEXT NOT NULL,
    diagnosis VARCHAR(100) NOT NULL
);

CREATE TABLE Medicines (
    medicine_id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    amount INTEGER NOT NULL,
    valid_from TIMESTAMP NOT NULL,
    valid_to TIMESTAMP NOT NULL
);

CREATE TABLE Prescriptions (
    conclusion_id INTEGER REFERENCES Conclusions(conclusion_id),
    medicine_id INTEGER REFERENCES Medicines(medicine_id),
    PRIMARY KEY (conclusion_id, medicine_id)
);
