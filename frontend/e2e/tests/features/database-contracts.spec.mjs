/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { test, expect, forEachDatabase } from '../../support/test-fixture.mjs';
import { getTableConfig } from '../../support/database-config.mjs';
import { parseDocument } from '../../support/categories/document.mjs';
import { filterSessionKeys, verifyColumnsForType, verifyHashFields, verifyMembers, verifyStringValue } from '../../support/categories/keyvalue.mjs';

function expectRowsContain(rows, expectedText) {
    const needle = String(expectedText).toLowerCase();
    const found = rows.some(row => row.some(cell => String(cell).toLowerCase().includes(needle)));
    expect(found, `Rows should contain "${expectedText}"`).toBe(true);
}

function expectStorageUnitsContain(actual, expected) {
    for (const item of expected) {
        expect(actual, `Storage units should include ${item}`).toContain(item);
    }
}

test.describe('Database Contracts', () => {
    forEachDatabase('sql', (db) => {
        const tableName = db.testTable.name;
        const tableConfig = getTableConfig(db, tableName);

        test('lists configured SQL storage units', async ({ whodb }) => {
            const tables = (await whodb.getTables()).filter(t => !t.match(/^test_table_\d+/));
            expectStorageUnitsContain(tables, db.expectedTables);
        });

        test('supports SQL table browse, sort, search, and pagination contract', async ({ whodb }) => {
            await whodb.data(tableName);
            await whodb.sortBy(db.testTable.idField);

            let tableData = await whodb.getTableData();
            expect(tableData.rows.length).toBeGreaterThan(0);
            expect(tableData.columns).toEqual(tableConfig.expectedColumns);
            expectRowsContain(tableData.rows, db.testTable.firstName);

            await whodb.searchTable(db.testTable.searchTerm);
            tableData = await whodb.getTableData();
            expect(tableData.rows.length).toBeGreaterThan(0);
            expectRowsContain(tableData.rows, db.testTable.searchTerm);

            if (db.features.pagination) {
                await whodb.setTablePageSize(1);
                await whodb.submitTable();
                tableData = await whodb.getTableData();
                expect(tableData.rows.length).toEqual(1);
            }

        });
    });

    forEachDatabase('document', (db) => {
        const tableName = db.testTable.name;
        const tableConfig = getTableConfig(db, tableName);

        test('lists configured document storage units', async ({ whodb }) => {
            const actual = await whodb.getTables();
            const expected = db.expectedIndices || db.expectedTables;
            expectStorageUnitsContain(actual, expected);
        });

        test('supports document browse, search, and pagination contract', async ({ whodb }) => {
            await whodb.data(tableName);

            let tableData = await whodb.getTableData();
            expect(tableData.rows.length).toBeGreaterThan(0);
            expect(tableData.columns).toEqual(tableConfig.expectedColumns);

            const documents = tableData.rows.map(row => parseDocument(row));
            const expectedDocument = tableConfig.testData.initial.find(doc =>
                doc[db.testTable.identifierField] === db.testTable.firstName
            );
            expect(expectedDocument, 'Fixture should define the first document').toBeDefined();
            expect(documents.some(doc =>
                doc[db.testTable.identifierField] === expectedDocument[db.testTable.identifierField] &&
                doc.email === expectedDocument.email
            )).toBe(true);

            await whodb.searchTable(db.testTable.searchTerm);
            tableData = await whodb.getTableData();
            expect(tableData.rows.length).toBeGreaterThan(0);
            expectRowsContain(tableData.rows, db.testTable.searchTerm);

            if (db.features.pagination) {
                await whodb.setTablePageSize(1);
                await whodb.submitTable();
                tableData = await whodb.getTableData();
                expect(tableData.rows.length).toEqual(1);
            }

        });
    });

    forEachDatabase('keyvalue', (db) => {
        test('lists configured key-value keys', async ({ whodb }) => {
            const keys = filterSessionKeys(await whodb.getTables());
            expectStorageUnitsContain(keys, db.expectedKeys);
        });

        for (const [key, keyConfig] of Object.entries(db.keyTypes)) {
            test(`supports ${keyConfig.type} data shape for ${key}`, async ({ whodb }) => {
                await whodb.data(key);
                const { columns, rows } = await whodb.getTableData();
                expect(rows.length).toBeGreaterThan(0);
                expect(columns).toEqual(keyConfig.expectedColumns);
                verifyColumnsForType(columns, keyConfig.type);

                if (keyConfig.expectedFields) {
                    verifyHashFields(rows, keyConfig.expectedFields);
                }
                if (keyConfig.expectedMembers) {
                    verifyMembers(rows, keyConfig.expectedMembers);
                }
                if (keyConfig.expectedValue) {
                    verifyStringValue(rows, keyConfig.expectedValue);
                }
            });
        }

        test('supports key-value search contract', async ({ whodb }) => {
            await whodb.data(db.testTable.name);
            await whodb.searchTable(db.testTable.searchTerm);

            const { rows } = await whodb.getTableData();
            expect(rows.length).toBeGreaterThan(0);
            expectRowsContain(rows, db.testTable.searchTerm);
        });
    });

    forEachDatabase('cache', (db) => {
        test('lists configured cache keys', async ({ whodb }) => {
            const keys = await whodb.getTables();
            expectStorageUnitsContain(keys, db.expectedKeys);
        });

        test('supports cache item browse and value contract', async ({ whodb }) => {
            await whodb.data(db.testTable.name);
            const { rows } = await whodb.getTableData();
            expect(rows.length).toBeGreaterThan(0);
            expectRowsContain(rows, db.testTable.testValues.original);

            await whodb.data('cache:homepage');
            const homepage = await whodb.getTableData();
            expectRowsContain(homepage.rows, 'featured_products');
            expectRowsContain(homepage.rows, 'Summer Sale');

            await whodb.data('counter:page_views');
            const counter = await whodb.getTableData();
            expectRowsContain(counter.rows, '42857');
        });
    });
});
